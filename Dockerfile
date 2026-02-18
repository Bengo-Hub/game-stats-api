# syntax=docker/dockerfile:1.6

FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./

RUN GOTOOLCHAIN=auto go mod download
COPY . .

# Build the API binary
RUN GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/game-stats-api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && addgroup -S app && adduser -S app -G app
WORKDIR /app

# Copy the compiled binary
COPY --from=builder /bin/game-stats-api /usr/local/bin/game-stats-api

# Copy scripts/fixtures so runtime migrations/fixtures are available
COPY --from=builder /app/scripts ./scripts

RUN chmod +x /usr/local/bin/game-stats-api || true
USER app
EXPOSE 4000
ENTRYPOINT ["/usr/local/bin/game-stats-api"]
