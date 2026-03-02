# ① ベースイメージ
FROM golang:1.22-alpine

# ② 作業ディレクトリ作成
WORKDIR /app

# ③ go.mod と go.sum 先にコピー（キャッシュ効率）
COPY go.mod ./
RUN go mod download

# ④ ソースコードコピー
COPY . .

# ⑤ ビルド
RUN go build -o app

# ⑥ 実行
CMD ["./app"]  