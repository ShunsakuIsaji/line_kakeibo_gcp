package bq

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
)

type BQdata struct {
	ReceiptID       string    `bigquery:"receipt_id"`
	LineUserID      string    `bigquery:"line_user_id"`
	CreatedAt       time.Time `bigquery:"created_at"`
	Date            string    `bigquery:"date"`
	TotalAmount     int       `bigquery:"total_amount"`
	ShopName        string    `bigquery:"shop_name"`
	ShopAddress     string    `bigquery:"shop_address"`
	Category        string    `bigquery:"category"`
	Memo            string    `bigquery:"memo"`
	Confidence      float64   `bigquery:"confidence"`
	EventJSONFile   string    `bigquery:"event_json_filename"`
	ReceiptFileName string    `bigquery:"receipt_filename"`
}

func InsertToBQ(ctx context.Context, projectID, datasetID, tableID string, data *BQdata) error {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to create BigQuery client: %w", err)
	}

	inserter := client.Dataset(datasetID).Table(tableID).Inserter()

	payload := []BQdata{*data}
	if err = inserter.Put(ctx, payload); err != nil {
		return fmt.Errorf("failed to put data to Bigquery; %w", err)
	}

	return nil
}
