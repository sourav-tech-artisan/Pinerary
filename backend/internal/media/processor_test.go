package media

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestMakeThumbnailBoundsAndFormat(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 1200, 600))
	source.Set(10, 10, color.RGBA{R: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}

	thumbnail, format, err := makeThumbnail(encoded.Bytes())
	if err != nil {
		t.Fatalf("makeThumbnail() error = %v", err)
	}
	if format != "png" {
		t.Fatalf("format = %q", format)
	}
	config, decodedFormat, err := image.DecodeConfig(bytes.NewReader(thumbnail))
	if err != nil {
		t.Fatalf("decode thumbnail: %v", err)
	}
	if decodedFormat != "jpeg" || config.Width != 640 || config.Height != 320 {
		t.Fatalf("thumbnail = %s %dx%d", decodedFormat, config.Width, config.Height)
	}
}
