package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSeedanceAssetClientPostAcceptsNumericErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"ResponseMetadata":{"Action":"CreateAsset","RequestId":"req-1"},
			"Error":{"Code":400,"Message":"invalid asset url"},
			"Result":null
		}`))
	}))
	defer server.Close()

	client := NewSeedanceAssetClient(server.URL, "test-key")
	var result SeedanceCreateAssetResult
	err := client.post("CreateAsset", map[string]string{"URL": "https://example.com/a.png"}, &result)
	if err == nil {
		t.Fatal("expected upstream error")
	}
	if got := err.Error(); got != "seedance upstream error: [400] invalid asset url" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestSeedanceAssetClientPostAcceptsNumericZeroErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"ResponseMetadata":{"Action":"CreateAsset","RequestId":"req-2"},
			"Error":{"Code":0,"Message":""},
			"Result":{"Id":"asset-123"}
		}`))
	}))
	defer server.Close()

	client := NewSeedanceAssetClient(server.URL, "test-key")
	id, err := client.CreateAsset("group-1", "https://example.com/a.png", "image", "demo")
	if err != nil {
		t.Fatalf("CreateAsset failed: %v", err)
	}
	if id != "asset-123" {
		t.Fatalf("expected asset-123, got %s", id)
	}
}

func TestSeedanceUpstreamErrorActive(t *testing.T) {
	if seedanceUpstreamErrorActive(&seedanceUpstreamError{Code: 0, Message: ""}) {
		t.Fatal("code 0 with empty message should not be active")
	}
	if !seedanceUpstreamErrorActive(&seedanceUpstreamError{Code: 500, Message: "failed"}) {
		t.Fatal("non-zero code should be active")
	}
	if !seedanceUpstreamErrorActive(&seedanceUpstreamError{Code: "0", Message: "still failed"}) {
		t.Fatal("message should make error active even when code is 0")
	}
}
