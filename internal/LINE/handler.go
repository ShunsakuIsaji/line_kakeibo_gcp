package line

import (
	"context"
	"fmt"
	"log"
	"time"

	gcs "github.com/ShunsakuIsaji/line_kakeibo_gcp/internal/GCS"

	"github.com/line/line-bot-sdk-go/v8/linebot"
)

type Handler struct {
	Bot        *linebot.Client
	BucketName string
}

func (h *Handler) HandleEventAPI(ctx context.Context, event *linebot.Event) error {

	if event.Type != linebot.EventTypeMessage {
		return nil
	}

	switch msg := event.Message.(type) {

	case *linebot.ImageMessage:

		log.Println("image received")

		content, err := h.Bot.GetMessageContent(msg.ID).Do()
		if err != nil {
			return err
		}
		defer content.Content.Close()

		now := time.Now().Format("20060102150405")
		filename := fmt.Sprintf("%s-%s.jpg", msg.ID, now)

		err = gcs.UploadToGCS(ctx, h.BucketName, filename, content.Content)
		if err != nil {
			return err
		}

		_, err = h.Bot.ReplyMessage(
			event.ReplyToken,
			linebot.NewTextMessage("レシート受け取ったで"),
		).Do()

		return err

	case *linebot.TextMessage:

		log.Printf("text received: %s", msg.Text)

		_, err := h.Bot.ReplyMessage(
			event.ReplyToken,
			linebot.NewTextMessage("テキストはまだ対応してないで"),
		).Do()

		return err

	default:
		linebot.NewTextMessage("今はまだ対応してないで")
		log.Printf("unsupported message type: %T", msg)
	}

	return nil
}
