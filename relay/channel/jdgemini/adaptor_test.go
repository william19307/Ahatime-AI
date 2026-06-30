package jdgemini

import (
	"net/http"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetRequestURLBuildsJDGenerateContentURL(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl:    "https://agentrs.jd.com/api/saas/openai-u",
			UpstreamModelName: "Gemini-2.5-pro",
		},
	}

	url, err := adaptor.GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://agentrs.jd.com/api/saas/openai-u/v1/models/Gemini-2.5-pro:generateContent", url)
}

func TestGetRequestURLBuildsJDStreamURL(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		IsStream: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl:    "https://agentrs.jd.com/api/saas/openai-u/v1",
			UpstreamModelName: "gemini-3.1-pro-preview",
		},
	}

	url, err := adaptor.GetRequestURL(info)

	require.NoError(t, err)
	require.Equal(t, "https://agentrs.jd.com/api/saas/openai-u/v1/models/gemini-3.1-pro-preview:streamGenerateContent?alt=sse", url)
}

func TestSetupRequestHeaderUsesXAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Request = &http.Request{Header: http.Header{"Content-Type": []string{"application/json"}}}

	headers := http.Header{}
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeChatCompletions,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey: "jd-key",
		},
	}

	err := (&Adaptor{}).SetupRequestHeader(c, &headers, info)

	require.NoError(t, err)
	require.Equal(t, "jd-key", headers.Get("x-api-key"))
	require.Empty(t, headers.Get("Authorization"))
}
