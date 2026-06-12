// Package jdseedance 实现京东 SaaS 中转 (agentrs.jd.com) 的豆包 Seedance 视频生成渠道。
//
// 京东使用自创"插件执行"协议, 与火山 Ark 原生协议不兼容:
//
//	提交: POST {base}/api/saas/plugin-u/v1/exec/Multimodal-live-video
//	      body: {"model": "...", "content": [...], "parameters": {...}}
//	      resp: {"result": {"task_id": "task-xxx", "status": "pending"}, "error": null}
//	查询: POST {base}/api/saas/plugin-u/v1/exec/query-task
//	      body: {"taskId": "task-xxx"}
//	      resp: {"task_status": "success", "content": [{"video_url": {"url": "..."}}],
//	             "usage": {"video_output": 108900, ...}, "parameters": {...}, "error": {...}}
//
// usage.video_output 为火山口径的 token 用量, 映射为 CompletionTokens 用于按量计费。
package jdseedance

import (
	"bytes"
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

// ============================
// Request / Response structures
// ============================

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
	Model      string         `json:"model"`
	Content    []ContentItem  `json:"content"`
	Parameters map[string]any `json:"parameters"`
}

// submitResponse 京东提交任务返回
type submitResponse struct {
	Result struct {
		TaskID  string `json:"task_id"`
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"result"`
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// queryResponse 京东查询任务返回
type queryResponse struct {
	TaskID     string `json:"task_id"`
	TaskStatus string `json:"task_status"`
	Content    []struct {
		ID       string   `json:"id"`
		VideoURL MediaURL `json:"video_url"`
	} `json:"content"`
	Parameters map[string]any `json:"parameters"`
	Usage      struct {
		HasVideoInput bool   `json:"has_video_input"`
		Resolution    string `json:"resolution"`
		VideoOutput   int    `json:"video_output"`
	} `json:"usage"`
	Error struct {
		Code    any    `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// metadataParams 允许用户通过顶层字段或 metadata 透传的生成参数。
// Content 支持通过 metadata.content 传入多模态素材 (reference_image / reference_video /
// reference_audio / first_frame / last_frame 等), 文本项会被忽略(以 prompt 为准)。
type metadataParams struct {
	Content       []ContentItem  `json:"content,omitempty"`
	Resolution    string         `json:"resolution,omitempty"`
	Ratio         string         `json:"ratio,omitempty"`
	Duration      *dto.IntValue  `json:"duration,omitempty"`
	Frames        *dto.IntValue  `json:"frames,omitempty"`
	Seed          *dto.IntValue  `json:"seed,omitempty"`
	GenerateAudio *dto.BoolValue `json:"generate_audio,omitempty"`
	Watermark     *dto.BoolValue `json:"watermark,omitempty"`
	CameraFixed   *dto.BoolValue `json:"camera_fixed,omitempty"`
	Draft         *dto.BoolValue `json:"draft,omitempty"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string

	cachedPayload *requestPayload // BuildRequestURL/Body 共享, 每个请求一个 adaptor 实例
}

// getPayload 构建(或返回缓存的)上游请求体。
func (a *TaskAdaptor) getPayload(c *gin.Context) (*requestPayload, error) {
	if a.cachedPayload != nil {
		return a.cachedPayload, nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	body, err := a.convertToRequestPayload(&req)
	if err != nil {
		return nil, err
	}
	a.cachedPayload = body
	return body, nil
}

// selectPlugin 按内容选择京东插件端点:
//
//	含视频/音频参考 -> Multimodal-live-video (仅 Seedance 2.0)
//	含图片         -> d-image-to-video
//	纯文本         -> d-text-to-video
func selectPlugin(p *requestPayload) string {
	hasImage := false
	for _, item := range p.Content {
		switch item.Type {
		case "video_url", "audio_url":
			return "Multimodal-live-video"
		case "image_url":
			hasImage = true
		}
	}
	if hasImage {
		return "d-image-to-video"
	}
	return "d-text-to-video"
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
	plugin := "Multimodal-live-video"
	if a.cachedPayload != nil { // BuildRequestBody 先于 DoRequest 执行, 此处已有缓存
		plugin = selectPlugin(a.cachedPayload)
	}
	return fmt.Sprintf("%s/api/saas/plugin-u/v1/exec/%s", strings.TrimSuffix(a.baseURL, "/"), plugin), nil
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
		body.Model = info.UpstreamModelName
	} else {
		info.UpstreamModelName = body.Model
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

	var jResp submitResponse
	if err := common.Unmarshal(responseBody, &jResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if jResp.Result.TaskID == "" {
		msg := jResp.Msg
		if msg == "" {
			msg = string(responseBody)
		}
		taskErr = service.TaskErrorWrapper(fmt.Errorf("jd upstream error: %s", msg), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = common.GetTimestamp()
	ov.Model = info.OriginModelName

	c.JSON(http.StatusOK, ov)
	return jResp.Result.TaskID, responseBody, nil
}

// FetchTask 查询任务状态。京东使用 POST query-task + {"taskId": ...}。
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/api/saas/plugin-u/v1/exec/query-task", strings.TrimSuffix(baseUrl, "/"))
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

// CancelTask 调用京东 Cancel-task 取消任务。由 /v1/video/generations/:task_id/cancel 路由触发。
func (a *TaskAdaptor) CancelTask(baseUrl, key, taskID, proxy string) error {
	uri := fmt.Sprintf("%s/api/saas/plugin-u/v1/exec/Cancel-task", strings.TrimSuffix(baseUrl, "/"))
	payload, err := common.Marshal(map[string]string{"taskId": taskID})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var jResp struct {
		Error struct {
			Code    any    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Result any `json:"result"`
	}
	if err := common.Unmarshal(body, &jResp); err != nil {
		return fmt.Errorf("unmarshal cancel response failed: %s", body)
	}
	if code := fmt.Sprintf("%v", jResp.Error.Code); code != "0" && code != "<nil>" && code != "" {
		return fmt.Errorf("jd cancel failed: %s", jResp.Error.Message)
	}
	return nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

// convertToRequestPayload 将统一视频请求转换为京东插件协议。
// 参数来源优先级: metadata 显式键 < 顶层字段 (duration/size/seconds)。
func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq) (*requestPayload, error) {
	r := requestPayload{
		Model:      req.Model,
		Content:    []ContentItem{},
		Parameters: map[string]any{},
	}

	// metadata 中的生成参数与多模态素材
	var mp metadataParams
	if err := taskcommon.UnmarshalMetadata(req.Metadata, &mp); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}

	// metadata.content 里的素材 (参考视频/音频/首尾帧等), 文本项丢弃, 以 prompt 为准
	for _, item := range mp.Content {
		if item.Type == "text" {
			continue
		}
		r.Content = append(r.Content, item)
	}

	// 顶层 images 参考图
	if req.HasImage() {
		for _, imgURL := range req.Images {
			r.Content = append(r.Content, ContentItem{
				Type:     "image_url",
				ImageURL: &MediaURL{URL: imgURL},
				Role:     "reference_image",
			})
		}
	}

	// 提示词
	r.Content = append(r.Content, ContentItem{Type: "text", Text: req.Prompt})
	if mp.Resolution != "" {
		r.Parameters["resolution"] = strings.ToLower(mp.Resolution)
	}
	if mp.Ratio != "" {
		r.Parameters["ratio"] = mp.Ratio
	}
	if mp.Duration != nil {
		r.Parameters["duration"] = int(*mp.Duration)
	}
	if mp.Frames != nil {
		r.Parameters["frames"] = int(*mp.Frames)
	}
	if mp.Seed != nil {
		r.Parameters["seed"] = int(*mp.Seed)
	}
	if mp.GenerateAudio != nil {
		r.Parameters["generate_audio"] = bool(*mp.GenerateAudio)
	}
	if mp.Watermark != nil {
		r.Parameters["watermark"] = bool(*mp.Watermark)
	}
	if mp.CameraFixed != nil {
		r.Parameters["camera_fixed"] = bool(*mp.CameraFixed)
	}
	if mp.Draft != nil {
		r.Parameters["draft"] = bool(*mp.Draft)
	}

	// 顶层字段透传 (优先级高于 metadata)
	if req.Duration > 0 {
		r.Parameters["duration"] = req.Duration
	}
	if sec, _ := strconv.Atoi(req.Seconds); sec > 0 {
		r.Parameters["duration"] = sec
	}
	if res := normalizeResolution(req.Size); res != "" {
		r.Parameters["resolution"] = res
	}

	return &r, nil
}

// normalizeResolution 把 "480p"/"720P"/"864x480"/"1280x720" 统一为京东接受的 "480p"/"720p"。
func normalizeResolution(size string) string {
	s := strings.ToLower(strings.TrimSpace(size))
	if s == "" {
		return ""
	}
	if strings.HasSuffix(s, "p") {
		return s
	}
	if idx := strings.LastIndexByte(s, 'x'); idx > 0 {
		if h, err := strconv.Atoi(s[idx+1:]); err == nil && h > 0 {
			return fmt.Sprintf("%dp", h)
		}
	}
	return ""
}

// ParseTaskResult 解析京东查询返回, 映射为内部任务状态。
func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var resTask queryResponse
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	// 京东业务级错误 (如 key 无效) 没有 task_status 字段
	if resTask.TaskStatus == "" {
		return nil, fmt.Errorf("jd upstream error: %s", resTask.Msg)
	}

	taskResult := relaycommon.TaskInfo{Code: 0}

	switch strings.ToLower(resTask.TaskStatus) {
	case "pending", "queued", "submitted", "waiting":
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = "10%"
	case "running", "processing", "in_progress", "generating":
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "50%"
	case "success", "succeeded", "succeed", "done", "finished", "completed":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		if len(resTask.Content) > 0 {
			taskResult.Url = resTask.Content[0].VideoURL.URL
		}
		// 计费 token 采用京东账单口径(与京东后台逐位对账):
		//   含视频输入: 原生 28元/M 档, token = video_output 原值
		//   无视频输入: 原生 46元/M 档, 京东按 0.028 标价展示, token = video_output × 46/28 后向下取整到 10
		// 模型单价请按 0.028元/千token(28元/M) × 毛利率设置。
		billingTokens := resTask.Usage.VideoOutput
		if !resTask.Usage.HasVideoInput {
			billingTokens = int(int64(resTask.Usage.VideoOutput) * 46 / 28 / 10 * 10)
		}
		taskResult.CompletionTokens = billingTokens
		taskResult.TotalTokens = billingTokens
	case "failed", "fail", "error", "expired", "cancelled", "canceled":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = resTask.Error.Message
	default:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "30%"
	}

	return &taskResult, nil
}

// ConvertToOpenAIVideo 把存储的京东任务数据转换为 OpenAI Video 对象 (查询接口返回)。
func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var jResp queryResponse
	if err := common.Unmarshal(originTask.Data, &jResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal jd seedance task data failed")
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = originTask.TaskID
	openAIVideo.TaskID = originTask.TaskID
	openAIVideo.Status = originTask.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(originTask.Progress)
	if len(jResp.Content) > 0 {
		openAIVideo.SetMetadata("url", jResp.Content[0].VideoURL.URL)
	}
	openAIVideo.CreatedAt = originTask.CreatedAt
	openAIVideo.CompletedAt = originTask.UpdatedAt
	openAIVideo.Model = originTask.Properties.OriginModelName

	if strings.EqualFold(jResp.TaskStatus, "failed") {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: jResp.Error.Message,
			Code:    fmt.Sprintf("%v", jResp.Error.Code),
		}
	}

	return common.Marshal(openAIVideo)
}
