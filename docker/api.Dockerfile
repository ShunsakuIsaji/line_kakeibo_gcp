# ① ベースイメージ
FROM golang:1.24-alpine AS builder

# ② 作業ディレクトリ作成
WORKDIR /app

# ③ go.mod と go.sum 先にコピー（キャッシュ効率）
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# ④ ソースコードコピー
COPY . .

# ⑤ ビルド
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api ./cmd/api

# ---------- runtime stage ----------
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# バイナリだけコピー
COPY --from=builder /app/api .

# Cloud RunはPORT環境変数使う
ENV PORT=8080

# 起動
CMD ["/app/api"]