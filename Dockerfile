FROM golang:alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN GOPROXY=https://goproxy.cn,direct go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o mcp-gateway ./cmd/gateway

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -u 1000 appuser

WORKDIR /app
COPY --from=builder /build/mcp-gateway .
COPY config/servers.example.json /app/config.json

RUN chown -R appuser:appuser /app

USER appuser
EXPOSE 4298

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:4298/health || exit 1

ENTRYPOINT ["./mcp-gateway"]
CMD ["--config", "/app/config.json"]
