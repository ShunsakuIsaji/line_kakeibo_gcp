FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o worker ./cmd/worker

# ---------- runtime stage ----------
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# バイナリだけコピー
COPY --from=builder /app/worker .

# Cloud RunはPORT環境変数使う
ENV PORT=8080

# 起動
CMD ["/app/worker"]