package service

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeSeedanceAssetType(t *testing.T) {
	cases := map[string]string{
		"image": "Image",
		"Image": "Image",
		"IMAGE": "Image",
		"video": "Video",
		"audio": "Audio",
	}
	for input, want := range cases {
		got, err := NormalizeSeedanceAssetType(input)
		if err != nil {
			t.Fatalf("NormalizeSeedanceAssetType(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("NormalizeSeedanceAssetType(%q) = %q, want %q", input, got, want)
		}
	}
	if _, err := NormalizeSeedanceAssetType("pdf"); err == nil {
		t.Fatal("expected invalid asset type error")
	}
}

func TestInferSeedanceAssetTypeFromMIME(t *testing.T) {
	got, err := InferSeedanceAssetTypeFromMIME("image/png")
	if err != nil || got != "Image" {
		t.Fatalf("expected Image, got %q err=%v", got, err)
	}
	got, err = InferSeedanceAssetTypeFromMIME("video/mp4")
	if err != nil || got != "Video" {
		t.Fatalf("expected Video, got %q err=%v", got, err)
	}
	if _, err := InferSeedanceAssetTypeFromMIME("application/pdf"); err == nil {
		t.Fatal("expected error for pdf mime")
	}
}

func TestValidateSeedanceImageDimensions(t *testing.T) {
	if err := validateSeedanceImageDimensions(300, 300); err != nil {
		t.Fatalf("expected 300x300 valid: %v", err)
	}
	if err := validateSeedanceImageDimensions(299, 400); err == nil {
		t.Fatal("expected too small width to fail")
	}
	if err := validateSeedanceImageDimensions(400, 7000); err == nil {
		t.Fatal("expected too tall image to fail")
	}
	if err := validateSeedanceImageDimensions(1000, 3000); err == nil {
		t.Fatal("expected aspect ratio failure")
	}
}

func seedanceTestPNG300(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestValidateSeedanceImageURL(t *testing.T) {
	pngData := seedanceTestPNG300(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngData)
	}))
	defer server.Close()

	if err := validateSeedanceImageURL(server.URL + "/ok.png"); err != nil {
		t.Fatalf("expected valid image url: %v", err)
	}
}

func TestFormatSeedanceUpstreamError(t *testing.T) {
	msg := FormatSeedanceUpstreamError(fmt.Errorf("seedance upstream error: [400] WidthTooSmall"))
	if !strings.Contains(msg, "300") || !strings.Contains(msg, "6000") {
		t.Fatalf("unexpected formatted message: %q", msg)
	}
}

func TestValidateSeedanceUploadMIME(t *testing.T) {
	if err := validateSeedanceUploadMIME("image/png"); err != nil {
		t.Fatalf("expected png allowed: %v", err)
	}
	if err := validateSeedanceUploadMIME("application/pdf"); err == nil {
		t.Fatal("expected pdf rejected")
	}
}

func TestValidateSeedanceAssetURLReachable(t *testing.T) {
	pngData := seedanceTestPNG300(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/missing") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pngData)
	}))
	defer server.Close()

	if err := validateSeedanceAssetURLReachable(server.URL + "/file.jpg"); err != nil {
		t.Fatalf("expected reachable url: %v", err)
	}
	if err := validateSeedanceAssetURLReachable(server.URL + "/missing"); err == nil {
		t.Fatal("expected unreachable for 404 response")
	}
}

func TestValidateSeedanceAssetURLUnreachable(t *testing.T) {
	err := validateSeedanceAssetURLReachable("http://127.0.0.1:1/not-exists")
	if err == nil {
		t.Fatal("expected unreachable")
	}
}
