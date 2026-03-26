package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	bq "github.com/ShunsakuIsaji/line_kakeibo_gcp/internal/bq"
	gcs "github.com/ShunsakuIsaji/line_kakeibo_gcp/internal/gcs"
	gemini "github.com/ShunsakuIsaji/line_kakeibo_gcp/internal/gemini"
	line "github.com/ShunsakuIsaji/line_kakeibo_gcp/internal/line"
	pubsub "github.com/ShunsakuIsaji/line_kakeibo_gcp/internal/pubsub"
	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v8/linebot"
)

type Handler struct {
	Bot             *linebot.Client
	ImageBucketName string
	JSONBucketName  string
	GcpProjectID    string
	PubSubTopicID   string
	GeminiEndpoint  string
	BqDatasetID     string
	BqTableID       string
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

		// GCSから画像をダウンロードする
		data, err := gcs.DownloadFromGCS(r.Context(), h.ImageBucketName, pubSubMsg.ImageFileName)
		if err != nil {
			log.Printf("failed to download image: %s", err)
			line.PushMessage(h.Bot, pubSubMsg.LineUserID, fmt.Sprintf("画像のダウンロードに失敗しました: %s", err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		log.Printf("Downloaded image data size: %d bytes", len(data))

		// Gemini APIに送信して結果を取得する

		geminiReqBody := gemini.GetGeminiRequestBody(base64.StdEncoding.EncodeToString(data))

		geminiResp, err := gemini.GetGeminiResponse(h.GeminiEndpoint, geminiReqBody)

		if err != nil {
			log.Printf("failed to unmarshal Gemini response: %s", err)
			line.PushMessage(h.Bot, pubSubMsg.LineUserID, fmt.Sprintf("Gemini APIのレスポンスの解析に失敗しました: %s", err))
			// エラーは返さない
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Printf("Gemini response: %+v", geminiResp)

		// Geminiのレスポンスが成功したら、StorageにJSONを保存する
		jsonFileName := strings.Replace(pubSubMsg.ImageFileName, ".jpg", ".json", 1)
		jsonData, _ := json.Marshal(geminiResp)
		err = gcs.UploadToGCS(r.Context(), h.JSONBucketName, jsonFileName, bytes.NewReader(jsonData))
		if err != nil {
			log.Printf("failed to upload JSON to GCS: %s", err)
			line.PushMessage(h.Bot, pubSubMsg.LineUserID, fmt.Sprintf("解析結果の保存に失敗しました: %s", err))
			// エラーは返さない
			w.WriteHeader(http.StatusOK)
			return
		}

		if geminiResp.TotalAmount == 0 {
			line.PushMessage(h.Bot, pubSubMsg.LineUserID, "レシートの解析に失敗しました。画像が不鮮明な可能性があります。")
			w.WriteHeader(http.StatusOK)
			return
		}

		// BigQueryに保存する
		bqData := setBQdata(&pubSubMsg, geminiResp)
		err = bq.InsertToBQ(r.Context(), h.GcpProjectID, h.BqDatasetID, h.BqTableID, bqData)
		if err != nil {
			log.Printf("failed to insert data to BigQuery: %s", err)
			line.PushMessage(h.Bot, pubSubMsg.LineUserID, fmt.Sprintf("データベースへの保存に失敗しました: %s", err))
			// エラーは返さない
			w.WriteHeader(http.StatusOK)
			return
		}

		// LINEに返信
		replyMsg := fmt.Sprintf("レシート情報:\n日付: %s\n合計金額: %d\n店舗名: %s\nカテゴリ: %s\nメモ: %s\n信頼度: %.2f",
			geminiResp.Date, geminiResp.TotalAmount, geminiResp.ShopName, geminiResp.Category, geminiResp.Memo, *geminiResp.Confidence)
		line.PushMessage(h.Bot, pubSubMsg.LineUserID, replyMsg)

	} else if pubSubMsg.MessageType == "text" {
		log.Printf("Received text query: %s", pubSubMsg.Query)
		line.PushMessage(h.Bot, pubSubMsg.LineUserID, fmt.Sprintf("テキストクエリを受け取りました: %s", pubSubMsg.Query))

	} else {
		log.Printf("Unknown message type: %s", pubSubMsg.MessageType)
		line.PushMessage(h.Bot, pubSubMsg.LineUserID, fmt.Sprintf("対応していないメッセージタイプです!: %s", pubSubMsg.MessageType))
		// エラーは返さない
	}

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
		Bot:             bot,
		ImageBucketName: os.Getenv("RECEIPT_BUCKET"),
		JSONBucketName:  os.Getenv("JSON_BUCKET"),
		GcpProjectID:    os.Getenv("GOOGLE_PROJECT_ID"),
		PubSubTopicID:   os.Getenv("API_PUBLISH_TOPIC_ID"),
		GeminiEndpoint:  os.Getenv("GEMINI_API_ENDPOINT") + os.Getenv("GEMINI_API_KEY"),
		BqDatasetID:     os.Getenv("BQ_DATASET_ID"),
		BqTableID:       os.Getenv("BQ_TABLE_ID"),
	}

	http.HandleFunc("/callback", handler.SubscriptionHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Worker is listening on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func setBQdata(pubSubMsg *pubsub.PubSubMessage, geminiResp *gemini.GeminiResponse) *bq.BQdata {
	return &bq.BQdata{
		ReceiptID:       pubSubMsg.ReceiptID,
		LineUserID:      pubSubMsg.LineUserID,
		CreatedAt:       pubSubMsg.CreatedAt,
		Date:            geminiResp.Date,
		TotalAmount:     geminiResp.TotalAmount,
		ShopName:        geminiResp.ShopName,
		ShopAddress:     safeGetString(geminiResp.ShopAddress),
		Category:        geminiResp.Category,
		Memo:            geminiResp.Memo,
		Confidence:      safeGetFloat64(geminiResp.Confidence),
		EventJSONFile:   fmt.Sprintf("%s.json", strings.Replace(pubSubMsg.ImageFileName, ".jpg", "", 1)),
		ReceiptFileName: pubSubMsg.ImageFileName,
	}
}

func safeGetString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func safeGetFloat64(f *float64) float64 {
	if f == nil {
		return 0.0
	}
	return *f
}
