package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestJDImageEndpointTypesAreImageOnly(t *testing.T) {
	endpoints := GetEndpointTypesByChannelType(constant.ChannelTypeJDImage, "gpt-image-2-G")

	require.Equal(t, []constant.EndpointType{constant.EndpointTypeImageGeneration}, endpoints)
}

func TestGPTImage2GIsImageGenerationModel(t *testing.T) {
	require.True(t, IsImageGenerationModel("gpt-image-2-G"))
}
