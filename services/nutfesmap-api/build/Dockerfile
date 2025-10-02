# services/nutfesmap-api/build/Dockerfile

# ---- Build stage ----
FROM golang:1.24.5 AS builder
WORKDIR /src

# 依存キャッシュ
COPY services/nutfesmap-api/go.mod services/nutfesmap-api/go.sum ./
RUN go mod download

# アプリ本体
COPY services/nutfesmap-api/ .

# 静的リンクでビルド（CGO無効）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/server ./cmd/server

# ---- Runtime stage ----
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /out/server /app/server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
