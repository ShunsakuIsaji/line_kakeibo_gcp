package pubsub

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/pubsub/v2"
)

type PubSubMessage struct {
	ReceiptID     string    `json:"receipt_id"`
	CreatedAt     time.Time `json:"created_at"`
	LineUserID    string    `json:"line_user_id"`
	MessageType   string    `json:"message_type"`
	Query         string    `json:"query"`
	ImageFileName string    `json:"image_file_name"`
}

func PublishMessage(ctx context.Context, projectID, topicID string, payload []byte) error {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("failed to publish message: %s", err)
		return err
	}
	defer client.Close()

	publisher := client.Publisher(topicID)

	res := publisher.Publish(ctx, &pubsub.Message{
		Data: payload,
	})

	_, err = res.Get(ctx)
	if err != nil {
		log.Fatalf("failed to publish message: %s", err)
		return err
	}
	return nil
}
