package jdimage

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct{}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := strings.TrimSuffix(strings.TrimSpace(info.ChannelBaseUrl), "/")
	if baseURL == "" {
		baseURL = "https://agentrs.jd.com"
	}

	switch info.RelayMode {
	case relayconstant.RelayModeImagesGenerations:
		return fmt.Sprintf("%s/api/saas/plugin-u/v1/exec/images-generations-G", baseURL), nil
	case relayconstant.RelayModeImagesEdits:
		return fmt.Sprintf("%s/api/saas/plugin-u/v1/exec/image-edits-G", baseURL), nil
	default:
		return "", fmt.Errorf("unsupported relay mode for JD Image: %d", info.RelayMode)
	}
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Content-Type", "application/json")
	req.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	return nil, errors.New("chat completions are not supported by JD Image")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("claude messages are not supported by JD Image")
}

func (a *Adaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("gemini generateContent is not supported by JD Image")
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("rerank is not supported by JD Image")
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("embeddings are not supported by JD Image")
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("audio is not supported by JD Image")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("responses API is not supported by JD Image")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	switch info.RelayMode {
	case relayconstant.RelayModeImagesGenerations:
		return convertGenerationRequest(request), nil
	case relayconstant.RelayModeImagesEdits:
		return convertEditRequest(c, request)
	default:
		return nil, fmt.Errorf("unsupported image relay mode: %d", info.RelayMode)
	}
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	return jdImageHandler(c, resp, info)
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}

func convertGenerationRequest(request dto.ImageRequest) map[string]any {
	payload := map[string]any{
		"prompt": request.Prompt,
		"n":      imageCount(request),
	}
	if request.Size != "" {
		payload["size"] = request.Size
	}
	if request.ResponseFormat != "" {
		payload["response_format"] = request.ResponseFormat
	}
	if len(request.Style) > 0 {
		payload["style"] = request.Style
	}
	return payload
}

func convertEditRequest(c *gin.Context, request dto.ImageRequest) (map[string]any, error) {
	payload := map[string]any{
		"model":    jdImageEditModel(request.Model),
		"prompt":   request.Prompt,
		"n":        imageCount(request),
		"logo_add": 0,
	}
	if request.Size != "" {
		payload["size"] = request.Size
	}
	if len(request.OutputFormat) > 0 {
		payload["output_format"] = request.OutputFormat
	} else if raw, ok := request.Extra["output_format"]; ok {
		payload["output_format"] = raw
	}
	if raw, ok := request.Extra["logo_add"]; ok {
		payload["logo_add"] = raw
	}

	images, err := jdEditImages(c, request)
	if err != nil {
		return nil, err
	}
	if len(images) > 0 {
		payload["images"] = images
	} else if len(request.Images) > 0 {
		payload["images"] = request.Images
	}
	return payload, nil
}

func imageCount(request dto.ImageRequest) int {
	if request.N != nil && *request.N > 0 {
		return int(*request.N)
	}
	return 1
}

func jdImageEditModel(model string) string {
	model = strings.TrimSpace(model)
	if strings.EqualFold(model, "gpt-image-2-G") {
		return "gpt-image-2"
	}
	if model == "" {
		return "gpt-image-2"
	}
	return model
}

func jdEditImages(c *gin.Context, request dto.ImageRequest) ([]map[string]string, error) {
	if len(request.Images) > 0 {
		return rawImagesToJDImages(request.Images)
	}
	if len(request.Image) > 0 {
		return rawImagesToJDImages(request.Image)
	}
	if c == nil || c.Request == nil || !strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
		return nil, nil
	}

	mf := c.Request.MultipartForm
	if mf == nil {
		form, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return nil, fmt.Errorf("failed to parse image edit form request: %w", err)
		}
		c.Request.MultipartForm = form
		c.Request.PostForm = url.Values(form.Value)
		mf = form
	}

	images := valueImagesToJDImages(mf.Value)
	fileImages, err := fileImagesToJDImages(mf.File)
	if err != nil {
		return nil, err
	}
	images = append(images, fileImages...)
	return images, nil
}

func rawImagesToJDImages(raw json.RawMessage) ([]map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var values []any
	if err := common.Unmarshal(raw, &values); err == nil {
		return anyImagesToJDImages(values), nil
	}
	var one any
	if err := common.Unmarshal(raw, &one); err != nil {
		return nil, err
	}
	return anyImagesToJDImages([]any{one}), nil
}

func anyImagesToJDImages(values []any) []map[string]string {
	images := make([]map[string]string, 0, len(values))
	for _, value := range values {
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				images = append(images, map[string]string{"image_url": strings.TrimSpace(v)})
			}
		case map[string]any:
			if imageURL := stringFromAny(v["image_url"]); imageURL != "" {
				images = append(images, map[string]string{"image_url": imageURL})
			} else if imageURL := stringFromAny(v["url"]); imageURL != "" {
				images = append(images, map[string]string{"image_url": imageURL})
			}
		}
	}
	return images
}

func valueImagesToJDImages(values map[string][]string) []map[string]string {
	var images []map[string]string
	for _, key := range []string{"image", "image[]", "image_url", "image_urls"} {
		for _, value := range values[key] {
			if strings.TrimSpace(value) != "" {
				images = append(images, map[string]string{"image_url": strings.TrimSpace(value)})
			}
		}
	}
	return images
}

func fileImagesToJDImages(files map[string][]*multipart.FileHeader) ([]map[string]string, error) {
	var fileHeaders []*multipart.FileHeader
	for fieldName, headers := range files {
		if fieldName == "image" || fieldName == "image[]" || strings.HasPrefix(fieldName, "image[") {
			fileHeaders = append(fileHeaders, headers...)
		}
	}

	images := make([]map[string]string, 0, len(fileHeaders))
	for _, header := range fileHeaders {
		file, err := header.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open image file %s: %w", header.Filename, err)
		}
		data, readErr := io.ReadAll(file)
		_ = file.Close()
		if readErr != nil {
			return nil, fmt.Errorf("failed to read image file %s: %w", header.Filename, readErr)
		}
		mimeType := header.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = http.DetectContentType(data)
		}
		images = append(images, map[string]string{
			"image_url": fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data)),
		})
	}
	return images, nil
}

func stringFromAny(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func jdImageHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)

	var simpleResp dto.SimpleResponse
	_ = common.Unmarshal(responseBody, &simpleResp)
	if oaiError := simpleResp.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, types.WithOpenAIError(types.OpenAIError{
			Message: string(responseBody),
			Type:    "jd_image_error",
		}, resp.StatusCode)
	}

	if isOpenAIImageResponse(responseBody) {
		service.IOCopyBytesGracefully(c, resp, responseBody)
		return &simpleResp.Usage, nil
	}

	imageResp, err := jdResponseToOpenAIImage(responseBody, info)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if len(imageResp.Data) == 0 {
		service.IOCopyBytesGracefully(c, resp, responseBody)
		return &simpleResp.Usage, nil
	}

	jsonResp, err := common.Marshal(imageResp)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	service.IOCopyBytesGracefully(c, resp, jsonResp)
	return &simpleResp.Usage, nil
}

func isOpenAIImageResponse(body []byte) bool {
	var payload struct {
		Data []dto.ImageData `json:"data"`
	}
	if common.Unmarshal(body, &payload) != nil || len(payload.Data) == 0 {
		return false
	}
	for _, item := range payload.Data {
		if item.Url != "" || item.B64Json != "" {
			return true
		}
	}
	return false
}

func jdResponseToOpenAIImage(body []byte, info *relaycommon.RelayInfo) (*dto.ImageResponse, error) {
	var raw any
	if err := common.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	created := int64(0)
	if info != nil && !info.StartTime.IsZero() {
		created = info.StartTime.Unix()
	}
	resp := &dto.ImageResponse{Created: created}
	seen := map[string]struct{}{}
	resp.Data = collectImageData(raw, seen)
	return resp, nil
}

func collectImageData(value any, seen map[string]struct{}) []dto.ImageData {
	var out []dto.ImageData
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			out = append(out, collectImageData(item, seen)...)
		}
	case map[string]any:
		if data := imageDataFromMap(v); data.Url != "" || data.B64Json != "" {
			key := data.Url + data.B64Json
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				out = append(out, data)
			}
		}
		for key, item := range v {
			if key == "image_urls" || key == "imageUrls" || key == "urls" {
				out = append(out, collectImageData(item, seen)...)
				continue
			}
			out = append(out, collectImageData(item, seen)...)
		}
	case string:
		if data := imageDataFromString(v); data.Url != "" || data.B64Json != "" {
			key := data.Url + data.B64Json
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				out = append(out, data)
			}
		}
	}
	return out
}

func imageDataFromMap(value map[string]any) dto.ImageData {
	for _, key := range []string{"url", "image_url", "imageUrl"} {
		if s := stringFromAny(value[key]); s != "" {
			return dto.ImageData{Url: s}
		}
	}
	for _, key := range []string{"b64_json", "b64Json", "base64", "b64_image", "image_base64"} {
		if s := stringFromAny(value[key]); s != "" {
			return dto.ImageData{B64Json: stripDataURLPrefix(s)}
		}
	}
	return dto.ImageData{}
}

func imageDataFromString(value string) dto.ImageData {
	value = strings.TrimSpace(value)
	if value == "" {
		return dto.ImageData{}
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return dto.ImageData{Url: value}
	}
	if strings.HasPrefix(value, "data:image/") {
		return dto.ImageData{B64Json: stripDataURLPrefix(value)}
	}
	return dto.ImageData{}
}

func stripDataURLPrefix(value string) string {
	if idx := strings.Index(value, ","); strings.HasPrefix(value, "data:") && idx >= 0 {
		return value[idx+1:]
	}
	return value
}
