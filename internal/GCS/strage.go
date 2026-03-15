package gcs

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
)

func UploadToGCS(ctx context.Context, bucket, filename string, r io.Reader) error {

	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	w := client.Bucket(bucket).Object(filename).NewWriter(ctx)

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}

	return w.Close()
}
