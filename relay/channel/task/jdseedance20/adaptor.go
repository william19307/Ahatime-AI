// Package jdseedance20 实现京东 Seedance 2.0 dance-create / dance-query 协议。
//
//	提交: POST {base}/api/saas/plugin-u/v1/exec/dance-create
//	      body: {"content":[],"generate_audio":true,"ratio":"16:9","duration":5,"watermark":false}
//	      resp: {"code":1,"data":"<task_id>","msg":"..."}
//	查询: POST {base}/api/saas/plugin-u/v1/exec/dance-query
//	      body: {"taskId":"..."}
//	      resp: {"code":1,"data":{"id":"...","status":"...","content":...},"msg":"..."}
package jdseedance20

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type ContentItem struct {
	Type     string    `json:"type,omitempty"`
	Text     string    `json:"text,omitempty"`
	ImageURL *MediaURL `json:"image_url,omitempty"`
	VideoURL *MediaURL `json:"video_url,omitempty"`
	AudioURL *MediaURL `json:"audio_url,omitempty"`
	Role     string    `json:"role,omitempty"`
}

type MediaURL struct {
	URL string `json:"url,omitempty"`
}

type requestPayload struct {
	Content       []ContentItem `json:"content"`
	GenerateAudio bool          `json:"generate_audio"`
	Ratio         string        `json:"ratio"`
	Duration      int           `json:"duration"`
	Watermark     bool          `json:"watermark"`
}

type envelopeResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

type queryTaskData struct {
	ID      string          `json:"id"`
	Status  string          `json:"status"`
	Content json.RawMessage `json:"content"`
	Usage   struct {
		HasVideoInput    bool `json:"has_video_input"`
		VideoOutput      int  `json:"video_output"`
		CompletionTokens int  `json:"completion_tokens"`
		TotalTokens      int  `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Code    any    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type metadataParams struct {
	Content       []ContentItem  `json:"content,omitempty"`
	Ratio         string         `json:"ratio,omitempty"`
	Duration      *dto.IntValue  `json:"duration,omitempty"`
	GenerateAudio *dto.BoolValue `json:"generate_audio,omitempty"`
	Watermark     *dto.BoolValue `json:"watermark,omitempty"`
}

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string

	cachedPayload *requestPayload
}

const seedanceAssetURLPrefix = "seedance_asset://"

func (a *TaskAdaptor) getPayload(c *gin.Context) (*requestPayload, error) {
	if a.cachedPayload != nil {
		return a.cachedPayload, nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	for i, imgURL := range req.Images {
		resolved, resolveErr := resolveSeedanceMediaURL(c, imgURL)
		if resolveErr != nil {
			return nil, resolveErr
		}
		req.Images[i] = resolved
	}
	body, err := a.convertToRequestPayload(&req)
	if err != nil {
		return nil, err
	}
	if err := resolvePayloadSeedanceAssets(c, body); err != nil {
		return nil, err
	}
	sortContentItems(body.Content)
	a.cachedPayload = body
	return body, nil
}

func resolveSeedanceMediaURL(c *gin.Context, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if !strings.HasPrefix(rawURL, seedanceAssetURLPrefix) {
		return rawURL, nil
	}
	userId := c.GetInt("id")
	if userId <= 0 {
		return "", errors.New("seedance asset reference requires authenticated user")
	}
	idStr := strings.TrimPrefix(rawURL, seedanceAssetURLPrefix)
	localID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || localID <= 0 {
		return "", fmt.Errorf("invalid seedance asset reference: %s", rawURL)
	}
	return service.NewSeedanceAssetService().ResolveAssetURLForRelay(userId, localID)
}

func resolvePayloadSeedanceAssets(c *gin.Context, payload *requestPayload) error {
	if payload == nil {
		return nil
	}
	for i := range payload.Content {
		item := &payload.Content[i]
		switch item.Type {
		case "image_url":
			if item.ImageURL != nil {
				resolved, err := resolveSeedanceMediaURL(c, item.ImageURL.URL)
				if err != nil {
					return err
				}
				item.ImageURL.URL = resolved
			}
		case "video_url":
			if item.VideoURL != nil {
				resolved, err := resolveSeedanceMediaURL(c, item.VideoURL.URL)
				if err != nil {
					return err
				}
				item.VideoURL.URL = resolved
			}
		case "audio_url":
			if item.AudioURL != nil {
				resolved, err := resolveSeedanceMediaURL(c, item.AudioURL.URL)
				if err != nil {
					return err
				}
				item.AudioURL.URL = resolved
			}
		}
	}
	return nil
}

var contentTypeOrder = map[string]int{
	"text":       0,
	"image_url":  1,
	"video_url":  2,
	"audio_url":  3,
}

func sortContentItems(items []ContentItem) {
	if len(items) < 2 {
		return
	}
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			left := contentTypeOrder[items[i].Type]
			right := contentTypeOrder[items[j].Type]
			if right < left {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/api/saas/plugin-u/v1/exec/dance-create", strings.TrimSuffix(a.baseURL, "/")), nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	body, err := a.getPayload(c)
	if err != nil {
		return nil, errors.Wrap(err, "convert request payload failed")
	}
	if info.IsModelMapped {
		_ = info.UpstreamModelName
	} else {
		info.UpstreamModelName = info.OriginModelName
	}
	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	taskID, err = parseCreateTaskID(responseBody)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = common.GetTimestamp()
	ov.Model = info.OriginModelName

	c.JSON(http.StatusOK, ov)
	return taskID, responseBody, nil
}

func parseCreateTaskID(responseBody []byte) (string, error) {
	var envelope envelopeResponse
	if err := common.Unmarshal(responseBody, &envelope); err != nil {
		return "", errors.Wrapf(err, "unmarshal create response failed: %s", responseBody)
	}
	if envelope.Code >= 1000 {
		msg := strings.TrimSpace(envelope.Msg)
		if msg == "" {
			msg = string(responseBody)
		}
		return "", fmt.Errorf("jd upstream error: %s", msg)
	}
	taskID := extractTaskIDFromData(envelope.Data)
	if taskID == "" {
		msg := strings.TrimSpace(envelope.Msg)
		if msg == "" {
			msg = string(responseBody)
		}
		return "", fmt.Errorf("jd upstream error: %s", msg)
	}
	return taskID, nil
}

func extractTaskIDFromData(data json.RawMessage) string {
	if len(data) == 0 || string(data) == "null" {
		return ""
	}
	var taskID string
	if err := common.Unmarshal(data, &taskID); err == nil {
		return strings.TrimSpace(taskID)
	}
	var obj struct {
		ID     string `json:"id"`
		TaskID string `json:"task_id"`
	}
	if err := common.Unmarshal(data, &obj); err == nil {
		if id := strings.TrimSpace(obj.TaskID); id != "" {
			return id
		}
		return strings.TrimSpace(obj.ID)
	}
	return ""
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/api/saas/plugin-u/v1/exec/dance-query", strings.TrimSuffix(baseUrl, "/"))
	payload, err := common.Marshal(map[string]string{"taskId": taskID})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq) (*requestPayload, error) {
	r := requestPayload{
		Content:       []ContentItem{},
		GenerateAudio: true,
		Ratio:         "16:9",
		Duration:      5,
		Watermark:     false,
	}

	var mp metadataParams
	if err := taskcommon.UnmarshalMetadata(req.Metadata, &mp); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}

	hasText := false
	for _, item := range mp.Content {
		if item.Type == "text" {
			hasText = true
		}
		r.Content = append(r.Content, item)
	}

	if strings.TrimSpace(req.Prompt) != "" && !hasText {
		r.Content = append([]ContentItem{{Type: "text", Text: req.Prompt}}, r.Content...)
	} else if strings.TrimSpace(req.Prompt) != "" && hasText {
		// metadata 已含 text 时以 metadata 为准；若 prompt 非空且首项不是 text，补到最前
		if len(r.Content) == 0 || r.Content[0].Type != "text" {
			r.Content = append([]ContentItem{{Type: "text", Text: req.Prompt}}, r.Content...)
		}
	}

	if req.HasImage() {
		for _, imgURL := range req.Images {
			r.Content = append(r.Content, ContentItem{
				Type:     "image_url",
				ImageURL: &MediaURL{URL: imgURL},
				Role:     "reference_image",
			})
		}
	}

	if len(r.Content) == 0 {
		return nil, errors.New("content is required")
	}

	if mp.Ratio != "" {
		r.Ratio = mp.Ratio
	}
	if mp.Duration != nil {
		r.Duration = int(*mp.Duration)
	}
	if mp.GenerateAudio != nil {
		r.GenerateAudio = bool(*mp.GenerateAudio)
	}
	if mp.Watermark != nil {
		r.Watermark = bool(*mp.Watermark)
	}
	if req.Duration > 0 {
		r.Duration = req.Duration
	}
	if sec, _ := strconv.Atoi(req.Seconds); sec > 0 {
		r.Duration = sec
	}

	return &r, nil
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	taskData, err := unwrapQueryTaskData(respBody)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(taskData.Status) == "" {
		return nil, fmt.Errorf("jd upstream error: empty task status")
	}

	taskResult := relaycommon.TaskInfo{Code: 0}
	switch strings.ToLower(taskData.Status) {
	case "pending", "queued", "submitted", "waiting":
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = "10%"
	case "running", "processing", "in_progress", "generating":
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "50%"
	case "success", "succeeded", "succeed", "done", "finished", "completed":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		taskResult.Url = extractVideoURL(taskData.Content)
		billingTokens := taskData.Usage.VideoOutput
		if billingTokens == 0 {
			billingTokens = taskData.Usage.CompletionTokens
		}
		if billingTokens == 0 {
			billingTokens = taskData.Usage.TotalTokens
		}
		if billingTokens > 0 && !taskData.Usage.HasVideoInput {
			billingTokens = int(int64(billingTokens) * 46 / 28 / 10 * 10)
		}
		taskResult.CompletionTokens = billingTokens
		taskResult.TotalTokens = billingTokens
	case "failed", "fail", "error", "expired", "cancelled", "canceled":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = taskData.Error.Message
	default:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "30%"
	}
	return &taskResult, nil
}

func unwrapQueryTaskData(respBody []byte) (*queryTaskData, error) {
	var envelope envelopeResponse
	if err := common.Unmarshal(respBody, &envelope); err != nil {
		return nil, errors.Wrap(err, "unmarshal query envelope failed")
	}
	if envelope.Code >= 1000 {
		msg := strings.TrimSpace(envelope.Msg)
		if msg == "" {
			msg = string(respBody)
		}
		return nil, fmt.Errorf("jd upstream error: %s", msg)
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		msg := strings.TrimSpace(envelope.Msg)
		if msg == "" {
			msg = "empty query data"
		}
		return nil, fmt.Errorf("jd upstream error: %s", msg)
	}
	var taskData queryTaskData
	if err := common.Unmarshal(envelope.Data, &taskData); err != nil {
		return nil, errors.Wrap(err, "unmarshal query task data failed")
	}
	return &taskData, nil
}

func extractVideoURL(content json.RawMessage) string {
	if len(content) == 0 {
		return ""
	}
	var items []struct {
		VideoURL MediaURL `json:"video_url"`
	}
	if err := common.Unmarshal(content, &items); err == nil && len(items) > 0 {
		return items[0].VideoURL.URL
	}
	var single struct {
		VideoURL json.RawMessage `json:"video_url"`
	}
	if err := common.Unmarshal(content, &single); err != nil {
		return ""
	}
	if len(single.VideoURL) == 0 {
		return ""
	}
	var urlObj MediaURL
	if err := common.Unmarshal(single.VideoURL, &urlObj); err == nil && urlObj.URL != "" {
		return urlObj.URL
	}
	var urlStr string
	if err := common.Unmarshal(single.VideoURL, &urlStr); err == nil {
		return urlStr
	}
	return ""
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	taskData, err := unwrapQueryTaskData(originTask.Data)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal jd seedance2 task data failed")
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = originTask.TaskID
	openAIVideo.TaskID = originTask.TaskID
	openAIVideo.Status = originTask.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(originTask.Progress)
	if url := extractVideoURL(taskData.Content); url != "" {
		openAIVideo.SetMetadata("url", url)
	}
	openAIVideo.CreatedAt = originTask.CreatedAt
	openAIVideo.CompletedAt = originTask.UpdatedAt
	openAIVideo.Model = originTask.Properties.OriginModelName

	if strings.EqualFold(taskData.Status, "failed") {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: taskData.Error.Message,
			Code:    fmt.Sprintf("%v", taskData.Error.Code),
		}
	}

	return common.Marshal(openAIVideo)
}
