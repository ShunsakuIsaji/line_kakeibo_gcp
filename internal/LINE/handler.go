package line

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	gcs "github.com/ShunsakuIsaji/line_kakeibo_gcp/internal/GCS"
	pubsub "github.com/ShunsakuIsaji/line_kakeibo_gcp/internal/pubsub"
	"github.com/jaevor/go-nanoid"
	"github.com/line/line-bot-sdk-go/v8/linebot"
)

type Handler struct {
	Bot           *linebot.Client
	BucketName    string
	GcpProjectID  string
	PubSubTopicID string
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

		genId, err := nanoid.Standard(10)
		if err != nil {
			return err
		}
		receiptID := genId()

		filename := fmt.Sprintf("%s.jpg", receiptID)
		err = gcs.UploadToGCS(ctx, h.BucketName, filename, content.Content)
		if err != nil {
			return err
		}

		publishMessage, err := getPublishMessageFromEvent(event, receiptID)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(publishMessage)
		if err != nil {
			return err
		}

		err = pubsub.PublishMessage(ctx, h.GcpProjectID, h.PubSubTopicID, payload)
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

		publishMessage, err := getPublishMessageFromEvent(event, "")
		if err != nil {
			return err
		}
		payload, err := json.Marshal(publishMessage)
		if err != nil {
			return err
		}

		err = pubsub.PublishMessage(ctx, h.GcpProjectID, h.PubSubTopicID, payload)
		if err != nil {
			return err
		}

		_, err = h.Bot.ReplyMessage(
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

func getPublishMessageFromEvent(event *linebot.Event, receiptID string) (*pubsub.PubSubMessage, error) {
	if event.Source == nil || event.Source.UserID == "" {
		return nil, fmt.Errorf("invalid event source")
	}
	switch msg := event.Message.(type) {

	case *linebot.ImageMessage:
		return &pubsub.PubSubMessage{
			CreatedAt:     time.Now(),
			LineUserID:    event.Source.UserID,
			MessageType:   "image",
			ImageFileName: fmt.Sprintf("%s.jpg", receiptID),
			ReceiptID:     receiptID,
		}, nil

	case *linebot.TextMessage:
		return &pubsub.PubSubMessage{
			CreatedAt:   time.Now(),
			LineUserID:  event.Source.UserID,
			MessageType: "text",
			Query:       msg.Text,
			ReceiptID:   receiptID,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported message type: %T", msg)
	}
}
