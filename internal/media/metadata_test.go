package media

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

// newTestImage creates a small 2x2 RGBA image.
func newTestImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, image.Black)
	img.Set(1, 0, image.White)
	img.Set(0, 1, image.Transparent)
	img.Set(1, 1, image.Transparent)

	return img
}

func newTestJPEG() image.Image {
	f, _ := os.Open("testdata/test.jpg")
	defer f.Close()

	img, _, _ := image.Decode(f)
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("failed to encode test JPEG: %v", err)
	}
	return buf.Bytes()
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test PNG: %v", err)
	}
	return buf.Bytes()
}

func TestStripJPEG(t *testing.T) {
	data := encodeJPEG(t, newTestJPEG())
	out, err := StripMetadata(data, "image/jpeg")
	if err != nil {
		t.Fatalf("StripMetadata failed: %v", err)
	}
	// Verify output is valid JPEG
	if _, err := jpeg.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("output is not valid JPEG: %v", err)
	}
}

func TestStripPNG(t *testing.T) {
	data := encodePNG(t, newTestImage())
	out, err := StripMetadata(data, "image/png")
	if err != nil {
		t.Fatalf("StripMetadata failed: %v", err)
	}
	// Verify output is valid PNG
	if _, err := png.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("output is not valid PNG: %v", err)
	}
}

func TestPassthroughGIF(t *testing.T) {
	data := []byte("GIF89a fake gif data")
	out, err := StripMetadata(data, "image/gif")
	if err != nil {
		t.Fatalf("StripMetadata failed: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Error("GIF data should pass through unchanged")
	}
}

func TestPassthroughVideo(t *testing.T) {
	for _, ct := range []string{"video/mp4", "video/webm"} {
		data := []byte("fake video data")
		out, err := StripMetadata(data, ct)
		if err != nil {
			t.Fatalf("StripMetadata(%s) failed: %v", ct, err)
		}
		if !bytes.Equal(out, data) {
			t.Errorf("%s data should pass through unchanged", ct)
		}
	}
}

func TestCorruptJPEG(t *testing.T) {
	_, err := StripMetadata([]byte("not a jpeg"), "image/jpeg")
	if err == nil {
		t.Error("expected error for corrupt JPEG data")
	}
}

func TestCorruptPNG(t *testing.T) {
	_, err := StripMetadata([]byte("not a png"), "image/png")
	if err == nil {
		t.Error("expected error for corrupt PNG data")
	}
}
