# Ollama LLM Deployment Guide for Text-to-SQL

## Overview

This guide covers deploying Ollama with the `duckdb-nsql:7b` model for converting natural language questions into SQL queries in the game-stats-api.

## Prerequisites

- Docker or Kubernetes cluster
- GPU support (recommended for performance, optional)
- 8GB+ RAM for the model
- Access to game-stats database schema

## Docker Deployment

### 1. Pull Ollama Image

```bash
docker pull ollama/ollama:latest
```

### 2. Run Ollama Container

**CPU-only**:
```bash
docker run -d \
  --name ollama \
  -p 11434:11434 \
  -v ollama_data:/root/.ollama \
  -e OLLAMA_KEEP_ALIVE=24h \
  ollama/ollama:latest
```

**With GPU (NVIDIA)**:
```bash
docker run -d \
  --name ollama \
  --gpus all \
  -p 11434:11434 \
  -v ollama_data:/root/.ollama \
  -e OLLAMA_KEEP_ALIVE=24h \
  ollama/ollama:latest
```

### 3. Download duckdb-nsql Model

```bash
# Enter container
docker exec -it ollama bash

# Pull model
ollama pull duckdb-nsql:7b

# Test model
ollama run duckdb-nsql:7b "SELECT all players from teams table"

# Exit container
exit
```

### 4. Verify Installation

```bash
curl http://localhost:11434/api/tags
```

Expected response:
```json
{
  "models": [
    {
      "name": "duckdb-nsql:7b",
      "modified_at": "2026-02-04T10:00:00Z",
      "size": 4096000000
    }
  ]
}
```

## Kubernetes Deployment

### 1. Create Namespace (if needed)

```bash
kubectl create namespace analytics
```

### 2. Create PersistentVolumeClaim

```yaml
# ollama-pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ollama-data
  namespace: analytics
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
  storageClassName: standard
```

```bash
kubectl apply -f ollama-pvc.yaml
```

### 3. Create Deployment

```yaml
# ollama-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ollama
  namespace: analytics
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ollama
  template:
    metadata:
      labels:
        app: ollama
    spec:
      containers:
      - name: ollama
        image: ollama/ollama:latest
        ports:
        - containerPort: 11434
          name: http
        env:
        - name: OLLAMA_KEEP_ALIVE
          value: "24h"
        - name: OLLAMA_NUM_PARALLEL
          value: "4"
        - name: OLLAMA_MAX_LOADED_MODELS
          value: "1"
        volumeMounts:
        - name: ollama-data
          mountPath: /root/.ollama
        resources:
          requests:
            memory: "8Gi"
            cpu: "2"
          limits:
            memory: "16Gi"
            cpu: "4"
        readinessProbe:
          httpGet:
            path: /api/tags
            port: 11434
          initialDelaySeconds: 30
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /api/tags
            port: 11434
          initialDelaySeconds: 60
          periodSeconds: 30
      volumes:
      - name: ollama-data
        persistentVolumeClaim:
          claimName: ollama-data
```

```bash
kubectl apply -f ollama-deployment.yaml
```

### 4. Create Service

```yaml
# ollama-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: ollama
  namespace: analytics
spec:
  selector:
    app: ollama
  ports:
  - name: http
    port: 11434
    targetPort: 11434
  type: ClusterIP
```

```bash
kubectl apply -f ollama-service.yaml
```

### 5. Load Model into Container

```bash
# Get pod name
kubectl get pods -n analytics | grep ollama

# Exec into pod
kubectl exec -it ollama-<pod-id> -n analytics -- bash

# Pull model
ollama pull duckdb-nsql:7b

# Verify
ollama list

# Exit
exit
```

### 6. Create ConfigMap for Schema (Optional)

```yaml
# schema-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: db-schema
  namespace: analytics
data:
  schema.sql: |
    -- Game Stats Database Schema
    CREATE TABLE events (
      id UUID PRIMARY KEY,
      name VARCHAR(255),
      start_date TIMESTAMP,
      end_date TIMESTAMP
    );
    
    CREATE TABLE teams (
      id UUID PRIMARY KEY,
      name VARCHAR(255),
      division_id UUID,
      seed INTEGER
    );
    
    CREATE TABLE players (
      id UUID PRIMARY KEY,
      first_name VARCHAR(100),
      last_name VARCHAR(100),
      team_id UUID
    );
    
    CREATE TABLE games (
      id UUID PRIMARY KEY,
      home_team_id UUID,
      away_team_id UUID,
      game_round_id UUID,
      home_score INTEGER,
      away_score INTEGER
    );
```

## Configuration for game-stats-api

### Environment Variables

Add to `.env` or Kubernetes Secret:

```env
OLLAMA_BASE_URL=http://ollama:11434
OLLAMA_MODEL=duckdb-nsql:7b
```

**Kubernetes Secret**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: game-stats-config
  namespace: analytics
type: Opaque
stringData:
  OLLAMA_BASE_URL: "http://ollama.analytics.svc.cluster.local:11434"
  OLLAMA_MODEL: "duckdb-nsql:7b"
```

### Update game-stats-api Deployment

```yaml
# Add to game-stats-api deployment
spec:
  template:
    spec:
      containers:
      - name: game-stats-api
        env:
        - name: OLLAMA_BASE_URL
          valueFrom:
            secretKeyRef:
              name: game-stats-config
              key: OLLAMA_BASE_URL
        - name: OLLAMA_MODEL
          valueFrom:
            secretKeyRef:
              name: game-stats-config
              key: OLLAMA_MODEL
```

## Testing the Integration

### 1. Health Check

```bash
curl http://localhost:11434/api/tags
```

### 2. Test SQL Generation

```bash
curl -X POST http://localhost:11434/api/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "duckdb-nsql:7b",
    "prompt": "Given the schema: teams(id, name), players(id, name, team_id). Write a SQL query to: Show me the top 5 teams by player count",
    "stream": false
  }'
```

### 3. Test via game-stats-api

```bash
curl -X POST http://localhost:4000/api/v1/analytics/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "question": "What are the top 5 teams by spirit score?",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "event_id": "650e8400-e29b-41d4-a716-446655440001"
  }'
```

## Performance Tuning

### Model Options

Adjust in `ollama_client.go`:

```go
"options": map[string]interface{}{
    "temperature": 0.2,  // Lower = more deterministic
    "top_p":       0.9,  // Nucleus sampling
    "num_ctx":     4096, // Context window size
    "num_predict": 512,  // Max tokens to generate
}
```

### Concurrent Requests

Set in Ollama environment:

```bash
OLLAMA_NUM_PARALLEL=4  # Handle 4 concurrent requests
OLLAMA_MAX_LOADED_MODELS=1  # Keep only duckdb-nsql loaded
```

### Caching Strategy

Implement Redis caching for common queries:

```go
// Pseudo-code
cacheKey := hash(question + schemaContext)
if cached := redis.Get(cacheKey); cached != nil {
    return cached
}
result := ollama.GenerateSQL(req)
redis.Set(cacheKey, result, 1*time.Hour)
```

## Monitoring

### Ollama Metrics

```bash
# Check model load status
curl http://localhost:11434/api/ps

# Expected response
{
  "models": [
    {
      "name": "duckdb-nsql:7b",
      "size": 4096000000,
      "digest": "sha256:...",
      "details": {
        "format": "gguf",
        "family": "llama"
      }
    }
  ]
}
```

### Application Metrics

Add Prometheus metrics:

```go
// In analytics service
var (
    queryLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name: "ollama_query_duration_seconds",
        Help: "Ollama query latency",
    }, []string{"status"})
    
    queriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "ollama_queries_total",
        Help: "Total Ollama queries",
    }, []string{"status"})
)
```

## Security Considerations

### 1. Input Validation
- Question length limit: 500 characters
- SQL validation: Block DELETE, UPDATE, DROP, etc.
- RLS enforcement: Always apply event_id filters

### 2. Rate Limiting
```go
// Implement rate limiter
limiter := rate.NewLimiter(rate.Every(time.Second), 10) // 10 req/s
if !limiter.Allow() {
    return errors.New("rate limit exceeded")
}
```

### 3. Network Policies

```yaml
# ollama-network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ollama-policy
  namespace: analytics
spec:
  podSelector:
    matchLabels:
      app: ollama
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: game-stats-api
    ports:
    - protocol: TCP
      port: 11434
```

## Troubleshooting

### Model Not Loading

```bash
# Check logs
docker logs ollama
# or
kubectl logs -n analytics ollama-<pod-id>

# Manually load model
docker exec -it ollama ollama pull duckdb-nsql:7b
```

### Out of Memory

Increase container memory limits:
```yaml
resources:
  limits:
    memory: "24Gi"  # Increase from 16Gi
```

### Slow Response Times

1. **Use GPU**: Reduces latency from 5-10s to 1-2s
2. **Reduce context**: Smaller schema prompts
3. **Cache results**: Redis for common queries
4. **Lower temperature**: 0.1-0.2 for faster generation

## Cost Optimization

### Cloud Deployment

**AWS EC2 (GPU)**:
- Instance: `g4dn.xlarge` (4 vCPU, 16GB RAM, T4 GPU)
- Cost: ~$0.526/hour = ~$380/month
- Performance: ~2s average query time

**AWS EC2 (CPU-only)**:
- Instance: `c5.2xlarge` (8 vCPU, 16GB RAM)
- Cost: ~$0.34/hour = ~$245/month
- Performance: ~8s average query time

### Scaling Strategy

1. **Start with CPU**: Test functionality
2. **Add GPU**: When query volume > 100/day
3. **Horizontal scaling**: Multiple replicas for > 1000 queries/day

## Backup & Recovery

### Model Backup

```bash
# Backup Ollama models
docker cp ollama:/root/.ollama ./ollama-backup

# Restore
docker cp ./ollama-backup ollama:/root/.ollama
```

### Kubernetes Backup

```bash
# Snapshot PVC
kubectl get pvc ollama-data -n analytics -o yaml > ollama-pvc-backup.yaml

# Create backup from PV
kubectl exec -n analytics ollama-<pod> -- tar czf /tmp/models.tar.gz /root/.ollama
kubectl cp analytics/ollama-<pod>:/tmp/models.tar.gz ./models-backup.tar.gz
```

## Next Steps

1. **pgvector Integration**: Add semantic search for schema context
2. **Query Optimization**: Analyze generated SQL, suggest indexes
3. **Multi-model Support**: Fallback models for high availability
4. **Fine-tuning**: Train on game-stats specific queries

## References

- Ollama Documentation: https://github.com/ollama/ollama
- duckdb-nsql Model: https://ollama.ai/library/duckdb-nsql
- Kubernetes Deployment: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/
