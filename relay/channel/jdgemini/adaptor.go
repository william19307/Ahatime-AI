package jdgemini

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/gemini"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct{}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := strings.TrimSuffix(strings.TrimSpace(info.ChannelBaseUrl), "/")
	if baseURL == "" {
		baseURL = "https://agentrs.jd.com/api/saas/openai-u"
	}
	baseURL = strings.TrimSuffix(baseURL, "/v1")

	modelName := strings.TrimSpace(info.UpstreamModelName)
	if modelName == "" {
		return "", errors.New("model name is empty")
	}

	action := "generateContent"
	if info.IsStream {
		action = "streamGenerateContent?alt=sse"
		if info.RelayMode == relayconstant.RelayModeGemini {
			info.DisablePing = true
		}
	}
	return fmt.Sprintf("%s/v1/models/%s:%s", baseURL, modelName, action), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Del("Authorization")
	req.Set("x-api-key", info.ApiKey)
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return gemini.CovertOpenAI2Gemini(c, *request, info)
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.ClaudeRequest) (any, error) {
	openAIAdaptor := openai.Adaptor{}
	converted, err := openAIAdaptor.ConvertClaudeRequest(c, info, req)
	if err != nil {
		return nil, err
	}
	openAIReq, ok := converted.(*dto.GeneralOpenAIRequest)
	if !ok {
		return nil, fmt.Errorf("unexpected converted request type: %T", converted)
	}
	return a.ConvertOpenAIRequest(c, info, openAIReq)
}

func (a *Adaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	for i := range request.Contents {
		if i == 0 && request.Contents[i].Role == "" {
			request.Contents[i].Role = "user"
		}
		for j := range request.Contents[i].Parts {
			part := &request.Contents[i].Parts[j]
			if part.FileData != nil && part.FileData.MimeType == "" && strings.Contains(part.FileData.FileUri, "www.youtube.com") {
				part.FileData.MimeType = "video/webm"
			}
		}
	}
	return request, nil
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("rerank is not supported by JD Gemini")
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("embeddings are not supported by JD Gemini")
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("audio is not supported by JD Gemini")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, errors.New("image generation is not supported by JD Gemini inference channel")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("responses API is not supported by JD Gemini")
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	if info.RelayMode == relayconstant.RelayModeGemini {
		if info.IsStream {
			return gemini.GeminiTextGenerationStreamHandler(c, info, resp)
		}
		return gemini.GeminiTextGenerationHandler(c, info, resp)
	}
	if info.IsStream {
		return gemini.GeminiChatStreamHandler(c, info, resp)
	}
	return gemini.GeminiChatHandler(c, info, resp)
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
