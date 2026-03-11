FROM golang:1.22-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o bin/checkstream ./cmd/server/...

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/bin/checkstream .
COPY --from=builder /app/config ./config
RUN mkdir -p settlements

EXPOSE 8080
CMD ["./checkstream"]
