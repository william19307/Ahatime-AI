package service

import (
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

func TestValidateSeedanceUploadMIME(t *testing.T) {
	if err := validateSeedanceUploadMIME("image/png"); err != nil {
		t.Fatalf("expected png allowed: %v", err)
	}
	if err := validateSeedanceUploadMIME("application/pdf"); err == nil {
		t.Fatal("expected pdf rejected")
	}
}

func TestValidateSeedanceAssetURLReachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/missing") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
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
