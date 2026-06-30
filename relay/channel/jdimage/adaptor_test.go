package jdimage

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetRequestURLBuildsJDImageURLs(t *testing.T) {
	adaptor := &Adaptor{}

	genURL, err := adaptor.GetRequestURL(&relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://agentrs.jd.com",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "https://agentrs.jd.com/api/saas/plugin-u/v1/exec/images-generations-G", genURL)

	editURL, err := adaptor.GetRequestURL(&relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesEdits,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://agentrs.jd.com/",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "https://agentrs.jd.com/api/saas/plugin-u/v1/exec/image-edits-G", editURL)
}

func TestSetupRequestHeaderUsesBearerJoyAgentKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Request = &http.Request{Header: http.Header{"Content-Type": []string{"multipart/form-data"}}}

	headers := http.Header{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ApiKey: "jd-key"},
	}

	err := (&Adaptor{}).SetupRequestHeader(c, &headers, info)

	require.NoError(t, err)
	require.Equal(t, "application/json", headers.Get("Content-Type"))
	require.Equal(t, "Bearer jd-key", headers.Get("Authorization"))
}

func TestConvertGenerationRequestOmitsModel(t *testing.T) {
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesGenerations}
	request := dto.ImageRequest{
		Model:          "gpt-image-2-G",
		Prompt:         "A cute baby sea otter",
		Size:           "1024x1024",
		N:              uintPtr(2),
		ResponseFormat: "url",
	}

	converted, err := (&Adaptor{}).ConvertImageRequest(nil, info, request)
	require.NoError(t, err)

	body, err := json.Marshal(converted)
	require.NoError(t, err)
	require.NotContains(t, string(body), `"model"`)
	require.Contains(t, string(body), `"prompt":"A cute baby sea otter"`)
	require.Contains(t, string(body), `"n":2`)
	require.Contains(t, string(body), `"response_format":"url"`)
}

func TestConvertJSONEditRequestMapsModelAndImages(t *testing.T) {
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesEdits}
	request := dto.ImageRequest{
		Model:        "gpt-image-2-G",
		Prompt:       "随意添加一朵红色玫瑰花",
		Size:         "1024x1024",
		N:            uintPtr(1),
		Images:       json.RawMessage(`[{"image_url":"https://example.com/input.png"}]`),
		OutputFormat: json.RawMessage(`"png"`),
	}

	converted, err := (&Adaptor{}).ConvertImageRequest(nil, info, request)
	require.NoError(t, err)

	body, err := json.Marshal(converted)
	require.NoError(t, err)
	require.Contains(t, string(body), `"model":"gpt-image-2"`)
	require.Contains(t, string(body), `"image_url":"https://example.com/input.png"`)
	require.Contains(t, string(body), `"output_format":"png"`)
	require.Contains(t, string(body), `"logo_add":0`)
}

func TestConvertMultipartEditRequestUsesDataURLImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2-G"))
	require.NoError(t, writer.WriteField("prompt", "add a flower"))
	require.NoError(t, writer.WriteField("size", "1024x1024"))
	part, err := writer.CreateFormFile("image", "input.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake image"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesEdits}
	request := dto.ImageRequest{Model: "gpt-image-2-G", Prompt: "add a flower", Size: "1024x1024"}

	converted, err := (&Adaptor{}).ConvertImageRequest(c, info, request)
	require.NoError(t, err)

	jsonBody, err := json.Marshal(converted)
	require.NoError(t, err)
	require.Contains(t, string(jsonBody), `"image_url":"data:`)
	require.Contains(t, string(jsonBody), `;base64,ZmFrZSBpbWFnZQ==`)
}

func TestDoResponseConvertsJDImageURLsToOpenAIImageResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesGenerations,
		StartTime: time.Unix(1700000000, 0),
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       ioNopCloser(`{"data":{"image_urls":["https://example.com/out.png"]}}`),
	}

	usage, err := (&Adaptor{}).DoResponse(c, resp, info)

	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Contains(t, recorder.Body.String(), `"url":"https://example.com/out.png"`)
	require.NotContains(t, recorder.Body.String(), "image_urls")
}

func TestDoResponseConvertsJDImageURLItems(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesEdits,
		StartTime: time.Unix(1700000000, 0),
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       ioNopCloser(`{"data":[{"image_url":"https://example.com/edit.png"}]}`),
	}

	_, err := (&Adaptor{}).DoResponse(c, resp, info)

	require.Nil(t, err)
	require.Contains(t, recorder.Body.String(), `"url":"https://example.com/edit.png"`)
	require.NotContains(t, recorder.Body.String(), "image_url")
}

type nopReadCloser struct {
	*strings.Reader
}

func (n nopReadCloser) Close() error {
	return nil
}

func ioNopCloser(body string) nopReadCloser {
	return nopReadCloser{Reader: strings.NewReader(body)}
}

func uintPtr(v uint) *uint {
	return &v
}
