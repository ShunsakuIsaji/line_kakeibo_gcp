package worker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	gcs "github.com/ShunsakuIsaji/line_kakeibo_gcp/internal/GCS"
	line "github.com/ShunsakuIsaji/line_kakeibo_gcp/internal/LINE"
	pubsub "github.com/ShunsakuIsaji/line_kakeibo_gcp/internal/pubsub"
	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v8/linebot"
)

type Handler struct {
	Bot           *linebot.Client
	BucketName    string
	GcpProjectID  string
	PubSubTopicID string
}

type SubscribedMessage struct {
	Message struct {
		Data string `json:"data"`
	} `json:"message"`
}

func (h *Handler) SubscriptionHandler(w http.ResponseWriter, r *http.Request) {

	var msg SubscribedMessage
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		log.Printf("failed to decode message: %s", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// base64 decode
	decoded, _ := base64.StdEncoding.DecodeString(msg.Message.Data)

	// 本体へ
	var pubSubMsg pubsub.PubSubMessage
	json.Unmarshal(decoded, &pubSubMsg)

	log.Printf("Received message: %+v", pubSubMsg)

	// image取得
	if pubSubMsg.MessageType == "image" {
		log.Printf("Image file name: %s", pubSubMsg.ImageFileName)
		// ここでCloud Storageから画像をダウンロードして処理することができます
		data, err := gcs.DownloadFromGCS(r.Context(), h.BucketName, pubSubMsg.ImageFileName)
		if err != nil {
			log.Printf("failed to download image: %s", err)
			line.PushMessage(h.Bot, pubSubMsg.LineUserID, fmt.Sprintf("画像のダウンロードに失敗しました: %s", err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		log.Printf("Downloaded image data size: %d bytes", len(data))
		// 例えば、画像解析やOCR処理などをここで行うことができます
	} else if pubSubMsg.MessageType == "text" {
		log.Printf("Received text query: %s", pubSubMsg.Query)
		line.PushMessage(h.Bot, pubSubMsg.LineUserID, fmt.Sprintf("テキストクエリを受け取りました: %s", pubSubMsg.Query))
		// 例えば、テキストクエリに対してAI処理を行い、結果をLINEに返信することができます
	}
	// 例えば、LINEへの返信や、Cloud Storageへの保存など

	w.WriteHeader(http.StatusOK)
}

func EnvLoad() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found; relying on injected env vars")
	}
}

func main() {
	// 環境変数の読み込み
	EnvLoad()

	// LINE Botクライアントの初期化
	bot, err := linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	handler := &Handler{
		Bot:           bot,
		BucketName:    os.Getenv("RECEIPT_BUCKET"),
		GcpProjectID:  os.Getenv("GOOGLE_PROJECT_ID"),
		PubSubTopicID: os.Getenv("API_PUBLISH_TOPIC_ID"),
	}

	http.HandleFunc("/callback", handler.SubscriptionHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Worker is listening on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
