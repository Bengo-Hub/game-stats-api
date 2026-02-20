#!/usr/bin/env bash
# =============================================================================
# Game Stats API Build & Deploy Script
# =============================================================================
# Purpose: Build Docker image, push to registry, and trigger ArgoCD deployment
# 
# Usage:
#   DEPLOY=true ./build.sh
#
# Environment Variables:
#   APP_NAME          - Application name (default: game-stats-api)
#   NAMESPACE         - Kubernetes namespace (default: mosuon)
#   DEPLOY            - Enable deployment (default: true)
#   SETUP_DATABASES   - Create database if needed (default: true)
#   SERVICE_DB_NAME   - Database name (default: game_stats)
#   SERVICE_DB_USER   - Database user (default: game_stats_user)
#   DEVOPS_REPO       - DevOps repository (default: Bengo-Hub/mosuon-devops-k8s)
#   DEVOPS_DIR        - Local devops directory (default: $HOME/mosuon-devops-k8s)
# =============================================================================

set -euo pipefail
set +H

# =============================================================================
# LOGGING
# =============================================================================
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# =============================================================================
# CONFIGURATION
# =============================================================================
APP_NAME=${APP_NAME:-"game-stats-api"}
NAMESPACE=${NAMESPACE:-"mosuon"}
ENV_SECRET_NAME=${ENV_SECRET_NAME:-"game-stats-api-secrets"}
DEPLOY=${DEPLOY:-true}
SETUP_DATABASES=${SETUP_DATABASES:-true}
SETUP_SECRETS=${SETUP_SECRETS:-true}
DB_TYPES=${DB_TYPES:-postgres,redis}

# Per-service database configuration
SERVICE_DB_NAME=${SERVICE_DB_NAME:-game_stats}
SERVICE_DB_USER=${SERVICE_DB_USER:-game_stats_user}
DB_NAMESPACE=${DB_NAMESPACE:-infra}

# Registry configuration
REGISTRY_SERVER=${REGISTRY_SERVER:-docker.io}
REGISTRY_NAMESPACE=${REGISTRY_NAMESPACE:-codevertex}
IMAGE_REPO="${REGISTRY_SERVER}/${REGISTRY_NAMESPACE}/${APP_NAME}"

# DevOps repository configuration
DEVOPS_REPO=${DEVOPS_REPO:-"Bengo-Hub/mosuon-devops-k8s"}
DEVOPS_DIR=${DEVOPS_DIR:-"$HOME/mosuon-devops-k8s"}
VALUES_FILE_PATH=${VALUES_FILE_PATH:-"apps/${APP_NAME}/values.yaml"}

# Git configuration
GIT_EMAIL=${GIT_EMAIL:-"dev@ultistats.ultichange.org"}
GIT_USER=${GIT_USER:-"Game Stats Bot"}
TRIVY_ECODE=${TRIVY_ECODE:-0}

# Determine Git commit ID
if [[ -z ${GITHUB_SHA:-} ]]; then
  GIT_COMMIT_ID=$(git rev-parse --short=8 HEAD || echo "localbuild")
else
  GIT_COMMIT_ID=${GITHUB_SHA::8}
fi

info "Service: ${APP_NAME}"
info "Namespace: ${NAMESPACE}"
info "Image: ${IMAGE_REPO}:${GIT_COMMIT_ID}"
info "Database: ${SERVICE_DB_NAME} (user: ${SERVICE_DB_USER})"

# =============================================================================
# PREREQUISITE CHECKS
# =============================================================================
for tool in git docker trivy; do
  command -v "$tool" >/dev/null || { error "$tool is required"; exit 1; }
done
if [[ ${DEPLOY} == "true" ]]; then
  for tool in kubectl helm yq jq; do
    command -v "$tool" >/dev/null || { error "$tool is required"; exit 1; }
  done
fi
success "Prerequisite checks passed"

# =============================================================================
# Auto-sync secrets from mosuon-devops-k8s
# =============================================================================
if [[ ${DEPLOY} == "true" ]]; then
  info "Checking and syncing required secrets from mosuon-devops-k8s..."
  SYNC_SCRIPT=$(mktemp)
  if curl -fsSL https://raw.githubusercontent.com/Bengo-Hub/mosuon-devops-k8s/master/scripts/tools/check-and-sync-secrets.sh -o "$SYNC_SCRIPT" 2>/dev/null; then
    source "$SYNC_SCRIPT"
    check_and_sync_secrets "REGISTRY_USERNAME" "REGISTRY_PASSWORD" "GIT_TOKEN" "POSTGRES_PASSWORD" "REDIS_PASSWORD" "KUBE_CONFIG" || warn "Secret sync failed - continuing with existing secrets"
    rm -f "$SYNC_SCRIPT"
  else
    warn "Unable to download secret sync script - continuing with existing secrets"
  fi
fi

# =============================================================================
# SECURITY SCAN
# =============================================================================
info "Running Trivy filesystem scan"
trivy fs . --exit-code "$TRIVY_ECODE" --format table || true

# =============================================================================
# BUILD DOCKER IMAGE
# =============================================================================
info "Building Docker image"
DOCKER_BUILDKIT=1 docker build . -t "${IMAGE_REPO}:${GIT_COMMIT_ID}" \
  --build-arg METABASE_BASE_URL=https://analytics.ultichange.org
success "Docker build complete"

if [[ ${DEPLOY} != "true" ]]; then
  warn "DEPLOY=false -> skipping push/deploy"
  exit 0
fi

# =============================================================================
# PUSH TO REGISTRY
# =============================================================================
if [[ -n ${REGISTRY_USERNAME:-} && -n ${REGISTRY_PASSWORD:-} ]]; then
  echo "$REGISTRY_PASSWORD" | docker login "$REGISTRY_SERVER" -u "$REGISTRY_USERNAME" --password-stdin
fi

docker push "${IMAGE_REPO}:${GIT_COMMIT_ID}"
success "Image pushed to ${IMAGE_REPO}:${GIT_COMMIT_ID}"

# =============================================================================
# KUBERNETES SETUP
# =============================================================================
if [[ -n ${KUBE_CONFIG:-} ]]; then
  mkdir -p ~/.kube
  echo "$KUBE_CONFIG" | base64 -d > ~/.kube/config
  chmod 600 ~/.kube/config
  export KUBECONFIG=~/.kube/config
fi

# Create namespace if needed
kubectl get ns "$NAMESPACE" >/dev/null 2>&1 || kubectl create ns "$NAMESPACE"

# Local dev secrets (not in CI)
if [[ -z ${CI:-}${GITHUB_ACTIONS:-} && -f KubeSecrets/devENV.yml ]]; then
  info "Applying local dev secrets"
  kubectl apply -n "$NAMESPACE" -f KubeSecrets/devENV.yml || warn "Failed to apply devENV.yml"
fi

# Create registry credentials
if [[ -n ${REGISTRY_USERNAME:-} && -n ${REGISTRY_PASSWORD:-} ]]; then
  kubectl -n "$NAMESPACE" create secret docker-registry registry-credentials \
    --docker-server="$REGISTRY_SERVER" \
    --docker-username="$REGISTRY_USERNAME" \
    --docker-password="$REGISTRY_PASSWORD" \
    --dry-run=client -o yaml | kubectl apply -f - || warn "Registry secret creation failed"
fi

# =============================================================================
# DATABASE SETUP (using centralized devops script)
# =============================================================================
if [[ "$SETUP_DATABASES" == "true" && -n "${KUBE_CONFIG:-}" ]]; then
  # Ensure devops repo is available
  if [[ ! -d "$DEVOPS_DIR" ]]; then
    info "Cloning mosuon-devops-k8s repository..."
    TOKEN="${GH_PAT:-${GIT_SECRET:-${GITHUB_TOKEN:-}}}"
    CLONE_URL="https://github.com/${DEVOPS_REPO}.git"
    [[ -n $TOKEN ]] && CLONE_URL="https://x-access-token:${TOKEN}@github.com/${DEVOPS_REPO}.git"
    git clone "$CLONE_URL" "$DEVOPS_DIR" || warn "Unable to clone devops repo"
  fi
  
  # Wait for PostgreSQL
  if kubectl -n "$DB_NAMESPACE" get statefulset postgresql >/dev/null 2>&1; then
    info "Waiting for PostgreSQL to be ready..."
    kubectl -n "$DB_NAMESPACE" rollout status statefulset/postgresql --timeout=180s || warn "PostgreSQL not fully ready"
    
    # Create database using centralized script
    if [[ -f "$DEVOPS_DIR/scripts/infrastructure/create-service-database.sh" ]]; then
      info "Creating database '${SERVICE_DB_NAME}' using centralized script..."
      chmod +x "$DEVOPS_DIR/scripts/infrastructure/create-service-database.sh"
      SERVICE_DB_NAME="$SERVICE_DB_NAME" \
      SERVICE_DB_USER="$SERVICE_DB_USER" \
      APP_NAME="$APP_NAME" \
      NAMESPACE="$NAMESPACE" \
      DB_NAMESPACE="$DB_NAMESPACE" \
      POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}" \
      bash "$DEVOPS_DIR/scripts/infrastructure/create-service-database.sh" || warn "Database creation failed or already exists"
    else
      warn "create-service-database.sh not found - using inline database creation"
      # Fallback to inline creation
      kubectl -n "$DB_NAMESPACE" exec -i postgresql-0 -- psql -U postgres <<EOF || warn "Database creation failed"
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_database WHERE datname = '${SERVICE_DB_NAME}') THEN
    CREATE DATABASE ${SERVICE_DB_NAME};
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${SERVICE_DB_USER}') THEN
    CREATE USER ${SERVICE_DB_USER} WITH PASSWORD '${POSTGRES_PASSWORD}';
  END IF;
  GRANT ALL PRIVILEGES ON DATABASE ${SERVICE_DB_NAME} TO ${SERVICE_DB_USER};
END
\$\$;
\c ${SERVICE_DB_NAME}
GRANT ALL ON SCHEMA public TO ${SERVICE_DB_USER};
EOF
    fi
    success "Database setup complete"
  else
    warn "PostgreSQL not found in ${DB_NAMESPACE} namespace - skipping database creation"
  fi
fi

# =============================================================================
# SECRETS SETUP (using centralized devops script)
# =============================================================================
if ! kubectl -n "$NAMESPACE" get secret "$ENV_SECRET_NAME" >/dev/null 2>&1; then
  if [[ -f "$DEVOPS_DIR/scripts/infrastructure/create-service-secrets.sh" ]]; then
    info "Creating secrets using centralized script..."
    chmod +x "$DEVOPS_DIR/scripts/infrastructure/create-service-secrets.sh"
    SERVICE_NAME="$APP_NAME" \
    NAMESPACE="$NAMESPACE" \
    DB_NAMESPACE="$DB_NAMESPACE" \
    DB_NAME="$SERVICE_DB_NAME" \
    DB_USER="$SERVICE_DB_USER" \
    SECRET_NAME="$ENV_SECRET_NAME" \
    POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}" \
    bash "$DEVOPS_DIR/scripts/infrastructure/create-service-secrets.sh" || warn "Secret creation failed"
  elif [[ -n "${POSTGRES_PASSWORD:-}" ]]; then
    info "Creating secrets inline..."
    DATABASE_URL="postgresql://${SERVICE_DB_USER}:${POSTGRES_PASSWORD}@postgresql.${DB_NAMESPACE}.svc.cluster.local:5432/${SERVICE_DB_NAME}?sslmode=disable"
    JWT_SECRET=${JWT_SECRET:-$(openssl rand -base64 32)}
    
    kubectl create secret generic "$ENV_SECRET_NAME" -n "$NAMESPACE" \
      --from-literal=DATABASE_URL="${DATABASE_URL}" \
      --from-literal=REDIS_PASSWORD="${POSTGRES_PASSWORD}" \
      --from-literal=JWT_SECRET="${JWT_SECRET}" \
      --dry-run=client -o yaml | kubectl apply -f -
    success "Secrets created"
  else
    warn "POSTGRES_PASSWORD not set and centralized script not available - secrets not created"
  fi
else
  info "Secret ${ENV_SECRET_NAME} already exists"
fi

# =============================================================================
# UPDATE HELM VALUES (using centralized script)
# =============================================================================
if [[ -f "${DEVOPS_DIR}/scripts/helm/update-values.sh" ]]; then
  info "Updating Helm values in devops repo..."
  chmod +x "${DEVOPS_DIR}/scripts/helm/update-values.sh"
  
  # Source and call the function
  source "${DEVOPS_DIR}/scripts/helm/update-values.sh" 2>/dev/null || true
  
  if declare -f update_helm_values >/dev/null 2>&1; then
    update_helm_values "$APP_NAME" "$GIT_COMMIT_ID" "$IMAGE_REPO"
    success "Helm values updated - ArgoCD will auto-sync"
  else
    # Direct script execution as fallback
    APP_NAME="$APP_NAME" \
    IMAGE_TAG="$GIT_COMMIT_ID" \
    IMAGE_REPO="$IMAGE_REPO" \
    DEVOPS_REPO="$DEVOPS_REPO" \
    DEVOPS_DIR="$DEVOPS_DIR" \
    VALUES_FILE_PATH="$VALUES_FILE_PATH" \
    GIT_EMAIL="$GIT_EMAIL" \
    GIT_USER="$GIT_USER" \
    bash "${DEVOPS_DIR}/scripts/helm/update-values.sh" --app "$APP_NAME" --tag "$GIT_COMMIT_ID" --repo "$IMAGE_REPO" || warn "Helm values update failed"
  fi
else
  warn "update-values.sh not found - manual Helm values update may be required"
fi

# =============================================================================
# SUMMARY
# =============================================================================
success "Build and deploy complete!"
echo ""
info "Deployment summary:"
echo "  Image      : ${IMAGE_REPO}:${GIT_COMMIT_ID}"
echo "  Namespace  : ${NAMESPACE}"
echo "  Database   : ${SERVICE_DB_NAME}"
echo "  Databases  : ${SETUP_DATABASES} (${DB_TYPES})"
echo ""
info "ArgoCD will auto-deploy ${APP_NAME}:${GIT_COMMIT_ID} to ${NAMESPACE} namespace"
