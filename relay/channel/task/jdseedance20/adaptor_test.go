package jdseedance20

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestParseCreateTaskID(t *testing.T) {
	id, err := parseCreateTaskID([]byte(`{"code":1,"data":"task-abc-123","msg":"ok"}`))
	if err != nil || id != "task-abc-123" {
		t.Fatalf("parseCreateTaskID = %q err=%v", id, err)
	}
}

func TestParseCreateTaskIDBizError(t *testing.T) {
	if _, err := parseCreateTaskID([]byte(`{"code":1001,"data":null,"msg":"请开通Api-Key使用"}`)); err == nil {
		t.Fatal("expected business error")
	}
}

func TestParseTaskResultSuccess(t *testing.T) {
	body := []byte(`{"code":1,"data":{"id":"task-1","status":"success","content":[{"video_url":{"url":"https://example.com/a.mp4"}}],"usage":{"has_video_input":false,"video_output":108900}},"msg":"ok"}`)
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if info.Url != "https://example.com/a.mp4" {
		t.Fatalf("url = %q", info.Url)
	}
	if info.CompletionTokens != 178900 {
		t.Fatalf("tokens = %d, want 178900", info.CompletionTokens)
	}
}

func TestParseTaskResultRunning(t *testing.T) {
	body := []byte(`{"code":1,"data":{"id":"task-1","status":"running"},"msg":"ok"}`)
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if info.Progress != "50%" {
		t.Fatalf("progress = %q", info.Progress)
	}
}

func TestConvertToRequestPayloadFlatFields(t *testing.T) {
	a := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "JDseedance2.0-10",
		Prompt:   "广告片",
		Duration: 11,
		Metadata: map[string]interface{}{
			"generate_audio": true,
			"ratio":          "16:9",
			"watermark":      false,
		},
	}
	p, err := a.convertToRequestPayload(&req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Duration != 11 || p.Ratio != "16:9" || !p.GenerateAudio || p.Watermark {
		t.Fatalf("unexpected payload: %+v", p)
	}
	if len(p.Content) != 1 || p.Content[0].Type != "text" {
		t.Fatalf("content = %+v", p.Content)
	}
}

func TestConvertToRequestPayloadReferenceMedia(t *testing.T) {
	a := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "JDseedance2.0-10",
		Prompt: "参考素材",
		Metadata: map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "video_url",
					"video_url": map[string]interface{}{
						"url": "https://example.com/ref.mp4",
					},
					"role": "reference_video",
				},
			},
			"ratio":    "16:9",
			"duration": 8,
		},
	}
	p, err := a.convertToRequestPayload(&req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(p.Content) != 2 {
		t.Fatalf("content len = %d", len(p.Content))
	}
	if p.Content[0].Type != "text" || p.Content[1].Type != "video_url" {
		t.Fatalf("content order = %+v", p.Content)
	}
}

func TestExtractVideoURLObject(t *testing.T) {
	url := extractVideoURL([]byte(`{"video_url":"https://example.com/a.mp4"}`))
	if url != "https://example.com/a.mp4" {
		t.Fatalf("url = %q", url)
	}
}
