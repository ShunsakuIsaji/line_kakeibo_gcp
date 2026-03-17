package main

import (
	"context"
	"log"
	"net/http"
	"os"

	line "github.com/ShunsakuIsaji/line_kakeibo_gcp/internal/LINE"
	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v8/linebot"
)

func main() {

	EnvLoad()

	bot, err := linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	handler := &line.Handler{
		Bot:           bot,
		BucketName:    os.Getenv("RECEIPT_BUCKET"),
		GcpProjectID:  os.Getenv("GOOGLE_PROJECT_ID"),
		PubSubTopicID: os.Getenv("API_PUBLISH_TOPIC_ID"),
	}

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {

		events, err := bot.ParseRequest(r)
		if err != nil {
			log.Println(err)
			w.WriteHeader(400)
			return
		}

		ctx := context.Background()

		for _, event := range events {

			err := handler.HandleEventAPI(ctx, event)
			if err != nil {
				log.Println(err)
			}

		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("start", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func EnvLoad() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}
