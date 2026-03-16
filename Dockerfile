FROM golang:1.23-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Bake git commit into binary so GET /health shows build_version (verify deployed build on Railway).
ARG RAILWAY_GIT_COMMIT_SHA
RUN BUILD_VERSION=${RAILWAY_GIT_COMMIT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo "local")} && \
    CGO_ENABLED=1 go build -ldflags "-X main.BuildVersion=$$BUILD_VERSION" -o bin/checkstream ./cmd/server/...

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/bin/checkstream .
COPY --from=builder /app/config ./config
RUN mkdir -p settlements

EXPOSE 8080
CMD ["./checkstream"]
