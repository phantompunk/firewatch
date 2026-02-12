package media

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"image/png"
)

// StripMetadata re-encodes images to remove EXIF, GPS, and other metadata.
// For unsupported types (GIF, WebP, video), data is returned unchanged.
func StripMetadata(data []byte, contentType string) ([]byte, error) {
	switch contentType {
	case "image/jpeg":
		return stripJPEG(data)
	case "image/png":
		return stripPNG(data)
	default:
		return data, nil
	}
}

func stripJPEG(data []byte) ([]byte, error) {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decoding jpeg: %w", err)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
		return nil, fmt.Errorf("encoding jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

func stripPNG(data []byte) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decoding png: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding png: %w", err)
	}
	return buf.Bytes(), nil
}
