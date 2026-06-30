package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
