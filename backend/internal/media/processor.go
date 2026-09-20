package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"path"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/objectstore"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	thumbnailMaxDimension = 640
	maxDecodedPixels      = 40_000_000
)

type Processor struct {
	store        objectstore.Store
	database     *dbgen.Queries
	maxPhotoSize int64
}

type photoJob struct {
	PhotoID uuid.UUID `json:"photo_id"`
}

func NewProcessor(database *dbgen.Queries, store objectstore.Store, maxPhotoSize int64) *Processor {
	return &Processor{database: database, store: store, maxPhotoSize: maxPhotoSize}
}

func (p *Processor) HandleJob(ctx context.Context, payload []byte) error {
	var job photoJob
	if err := json.Unmarshal(payload, &job); err != nil {
		return fmt.Errorf("decode photo processing job: %w", err)
	}
	if job.PhotoID == uuid.Nil {
		return fmt.Errorf("decode photo processing job: photo ID is required")
	}

	photo, err := p.database.GetPhotoByID(ctx, pgUUID(job.PhotoID))
	if err != nil {
		return fmt.Errorf("get photo for processing: %w", err)
	}
	if photo.Status == "processed" {
		return nil
	}

	reader, err := p.store.Get(ctx, photo.ObjectKey)
	if err != nil {
		return err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, p.maxPhotoSize+1))
	if err != nil {
		return fmt.Errorf("read uploaded photo: %w", err)
	}
	if int64(len(data)) > p.maxPhotoSize {
		return p.failPermanently(ctx, job.PhotoID)
	}

	thumbnail, format, err := makeThumbnail(data)
	if err != nil || !formatMatchesContentType(format, photo.ContentType) {
		return p.failPermanently(ctx, job.PhotoID)
	}
	thumbnailKey := path.Join(path.Dir(photo.ObjectKey), "thumbnail.jpg")
	if err := p.store.Put(ctx, thumbnailKey, bytes.NewReader(thumbnail), int64(len(thumbnail)), "image/jpeg"); err != nil {
		return err
	}
	_, err = p.database.MarkPhotoProcessed(ctx, dbgen.MarkPhotoProcessedParams{
		ID: pgUUID(job.PhotoID), ThumbnailKey: pgtype.Text{String: thumbnailKey, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("mark photo processed: %w", err)
	}
	return nil
}

func (p *Processor) failPermanently(ctx context.Context, photoID uuid.UUID) error {
	if err := p.database.MarkPhotoFailed(ctx, pgUUID(photoID)); err != nil {
		return fmt.Errorf("mark invalid photo failed: %w", err)
	}
	return nil
}

func makeThumbnail(data []byte) ([]byte, string, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode image header: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxDecodedPixels {
		return nil, "", fmt.Errorf("image dimensions are unsafe")
	}
	source, decodedFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}
	if decodedFormat != format {
		return nil, "", fmt.Errorf("image format changed while decoding")
	}

	width, height := scaledDimensions(config.Width, config.Height, thumbnailMaxDimension)
	destination := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(destination, destination.Bounds(), source, source.Bounds(), draw.Over, nil)

	var output bytes.Buffer
	if err := jpeg.Encode(&output, destination, &jpeg.Options{Quality: 82}); err != nil {
		return nil, "", fmt.Errorf("encode thumbnail: %w", err)
	}
	return output.Bytes(), format, nil
}

func scaledDimensions(width, height, maximum int) (int, int) {
	if width <= maximum && height <= maximum {
		return width, height
	}
	if width >= height {
		return maximum, max(1, height*maximum/width)
	}
	return max(1, width*maximum/height), maximum
}

func formatMatchesContentType(format, contentType string) bool {
	return (format == "jpeg" && contentType == "image/jpeg") ||
		(format == "png" && contentType == "image/png") ||
		(format == "webp" && contentType == "image/webp")
}
