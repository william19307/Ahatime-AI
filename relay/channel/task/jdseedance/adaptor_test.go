package jdseedance

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// 使用用户实测的京东真实返回验证解析逻辑
func TestParseTaskResultSuccess(t *testing.T) {
	body := []byte(`{"content":[{"id":"","video_url":{"url":"https://example.com/a.mp4"}}],
		"error":{"code":0,"message":"","type":""},
		"parameters":{"duration":5,"framespersecond":24,"ratio":"16:9","resolution":"720p"},
		"task_id":"task-b19xx9ifr4z1g8r","task_status":"success",
		"usage":{"has_video_input":false,"resolution":"720p","video_output":108900}}`)
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if info.Url != "https://example.com/a.mp4" {
		t.Fatalf("url = %q", info.Url)
	}
	// 无视频输入: 108900 × 46/28 = 178907.1 -> 向下取整到10 = 178900 (京东账单口径)
	if info.CompletionTokens != 178900 || info.TotalTokens != 178900 {
		t.Fatalf("tokens = %d/%d, want 178900", info.CompletionTokens, info.TotalTokens)
	}
}

// 用户实测样本: 无输入 100858 -> 京东账单 165690; 含输入 100858 -> 100858
func TestBillingTokensJDLedger(t *testing.T) {
	a := &TaskAdaptor{}
	noInput := []byte(`{"task_status":"success","content":[{"video_url":{"url":"https://x/v.mp4"}}],
		"usage":{"has_video_input":false,"resolution":"480p","video_output":100858},"error":{"code":0}}`)
	info, err := a.ParseTaskResult(noInput)
	if err != nil {
		t.Fatal(err)
	}
	if info.CompletionTokens != 165690 {
		t.Fatalf("no-input tokens = %d, want 165690", info.CompletionTokens)
	}
	noInput15s := []byte(`{"task_status":"success","content":[{"video_url":{"url":"https://x/v.mp4"}}],
		"usage":{"has_video_input":false,"resolution":"720p","video_output":324900},"error":{"code":0}}`)
	info, _ = a.ParseTaskResult(noInput15s)
	if info.CompletionTokens != 533760 {
		t.Fatalf("no-input 15s tokens = %d, want 533760", info.CompletionTokens)
	}
	withInput := []byte(`{"task_status":"success","content":[{"video_url":{"url":"https://x/v.mp4"}}],
		"usage":{"has_video_input":true,"resolution":"480p","video_output":100858},"error":{"code":0}}`)
	info, _ = a.ParseTaskResult(withInput)
	if info.CompletionTokens != 100858 {
		t.Fatalf("with-input tokens = %d, want 100858", info.CompletionTokens)
	}
}

func TestParseTaskResultRunning(t *testing.T) {
	body := []byte(`{"task_status":"running","task_id":"task-x","error":{"code":0},"parameters":null}`)
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if info.Progress != "50%" {
		t.Fatalf("progress = %q", info.Progress)
	}
}

func TestParseTaskResultBizError(t *testing.T) {
	body := []byte(`{"code":1001,"data":null,"msg":"请开通Api-Key使用"}`)
	a := &TaskAdaptor{}
	if _, err := a.ParseTaskResult(body); err == nil {
		t.Fatal("expected error for business error response")
	}
}

func TestConvertToRequestPayloadParams(t *testing.T) {
	a := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "Doubao-Seedance-2.0",
		Prompt:   "一只柴犬奔跑",
		Duration: 10,
		Size:     "480p",
		Metadata: map[string]interface{}{"generate_audio": false, "ratio": "9:16"},
	}
	p, err := a.convertToRequestPayload(&req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Parameters["duration"] != 10 {
		t.Fatalf("duration = %v", p.Parameters["duration"])
	}
	if p.Parameters["resolution"] != "480p" {
		t.Fatalf("resolution = %v", p.Parameters["resolution"])
	}
	if p.Parameters["ratio"] != "9:16" {
		t.Fatalf("ratio = %v", p.Parameters["ratio"])
	}
	if p.Parameters["generate_audio"] != false {
		t.Fatalf("generate_audio = %v", p.Parameters["generate_audio"])
	}
	if len(p.Content) != 1 || p.Content[0].Text != "一只柴犬奔跑" {
		t.Fatalf("content = %+v", p.Content)
	}
}

func TestNormalizeResolution(t *testing.T) {
	cases := map[string]string{"480p": "480p", "720P": "720p", "1280x720": "720p", "864x480": "480p", "": ""}
	for in, want := range cases {
		if got := normalizeResolution(in); got != want {
			t.Fatalf("normalizeResolution(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestConvertToRequestPayloadReferenceVideo(t *testing.T) {
	a := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "Doubao-Seedance-2.0",
		Prompt: "参考运镜",
		Metadata: map[string]interface{}{
			"duration": 5,
			"content": []interface{}{
				map[string]interface{}{
					"type":      "video_url",
					"video_url": map[string]interface{}{"url": "https://example.com/ref.mp4"},
					"role":      "reference_video",
				},
				map[string]interface{}{"type": "text", "text": "应被丢弃"},
			},
		},
	}
	p, err := a.convertToRequestPayload(&req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(p.Content) != 2 {
		t.Fatalf("content len = %d, want 2 (video + prompt text)", len(p.Content))
	}
	if p.Content[0].Type != "video_url" || p.Content[0].VideoURL.URL != "https://example.com/ref.mp4" || p.Content[0].Role != "reference_video" {
		t.Fatalf("video item = %+v", p.Content[0])
	}
	if p.Content[1].Type != "text" || p.Content[1].Text != "参考运镜" {
		t.Fatalf("text item = %+v", p.Content[1])
	}
	if p.Parameters["duration"] != 5 {
		t.Fatalf("duration = %v", p.Parameters["duration"])
	}
}

func TestSelectPlugin(t *testing.T) {
	text := &requestPayload{Content: []ContentItem{{Type: "text", Text: "x"}}}
	if got := selectPlugin(text); got != "d-text-to-video" {
		t.Fatalf("text -> %s", got)
	}
	img := &requestPayload{Content: []ContentItem{{Type: "image_url"}, {Type: "text"}}}
	if got := selectPlugin(img); got != "d-image-to-video" {
		t.Fatalf("image -> %s", got)
	}
	vid := &requestPayload{Content: []ContentItem{{Type: "image_url"}, {Type: "video_url"}, {Type: "text"}}}
	if got := selectPlugin(vid); got != "Multimodal-live-video" {
		t.Fatalf("video -> %s", got)
	}
	aud := &requestPayload{Content: []ContentItem{{Type: "audio_url"}, {Type: "text"}}}
	if got := selectPlugin(aud); got != "Multimodal-live-video" {
		t.Fatalf("audio -> %s", got)
	}
}
