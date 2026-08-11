package holycrab

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRelayInfo(modelName string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: modelName,
		},
	}
}

func TestConvertToRequestPayloadDefaults(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Prompt: "A cinematic product reveal",
		Size:   "720p",
	}

	payload, err := adaptor.convertToRequestPayload(&req, testRelayInfo("seedance-2-0"))

	require.NoError(t, err)
	assert.Equal(t, "A cinematic product reveal", payload.Prompt)
	assert.Equal(t, 5, payload.Duration)
	assert.Equal(t, "seedance-2-0", payload.Model)
	assert.Equal(t, "720p", payload.Resolution)
}

func TestConvertToRequestPayloadMetadataOverrides(t *testing.T) {
	adaptor := &TaskAdaptor{}
	generateAudio := true
	req := relaycommon.TaskSubmitReq{
		Prompt:   "Animate this scene",
		Duration: 8,
		Size:     "1080p",
		Metadata: map[string]any{
			"ratio":           "16:9",
			"generate_audio":  generateAudio,
			"videoAssetIds":   []any{"assetVideo1"},
			"imageAssetIds":   []any{"assetImage1"},
			"firstFrameAssetId": "assetFirst1",
		},
	}

	payload, err := adaptor.convertToRequestPayload(&req, testRelayInfo("seedance-2-0-fast"))

	require.NoError(t, err)
	assert.Equal(t, 8, payload.Duration)
	assert.Equal(t, "1080p", payload.Resolution)
	assert.Equal(t, "16:9", payload.Ratio)
	require.NotNil(t, payload.GenerateAudio)
	assert.True(t, *payload.GenerateAudio)
	assert.Equal(t, []string{"assetVideo1"}, payload.VideoAssetIds)
	assert.Equal(t, []string{"assetImage1"}, payload.ImageAssetIds)
	assert.Equal(t, "assetFirst1", payload.FirstFrameAssetId)
}

func TestParseTaskResultCompleted(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"code": 200,
		"data": {
			"uniqId": "abc123task456",
			"step": 2,
			"videoUrl": "https://cdn.example.com/video.mp4"
		},
		"message": null
	}`)

	taskInfo, err := adaptor.ParseTaskResult(body)

	require.NoError(t, err)
	assert.Equal(t, "abc123task456", taskInfo.TaskID)
	assert.Equal(t, "https://cdn.example.com/video.mp4", taskInfo.Url)
	assert.Equal(t, "100%", taskInfo.Progress)
}

func TestParseTaskResultFailed(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"code": 200,
		"data": {
			"uniqId": "abc123task456",
			"step": 3,
			"error": "余额不足"
		},
		"message": null
	}`)

	taskInfo, err := adaptor.ParseTaskResult(body)

	require.NoError(t, err)
	assert.Equal(t, "余额不足", taskInfo.Reason)
	assert.Equal(t, "100%", taskInfo.Progress)
}

func TestParseResolutionAndRatio(t *testing.T) {
	assert.Equal(t, "720p", parseResolution("1280x720"))
	assert.Equal(t, "1080p", parseResolution("1920x1080"))
	assert.Equal(t, "480p", parseResolution("512x512"))
	assert.Equal(t, "4k", parseResolution("3840x2160"))
	assert.Equal(t, "16:9", parseRatio("1280x720"))
	assert.Equal(t, "9:16", parseRatio("1080x1920"))
}

func TestTaskDurationUsesSecondsFallback(t *testing.T) {
	req := relaycommon.TaskSubmitReq{Seconds: "10"}
	assert.Equal(t, 10, taskDuration(req))
}

func TestNormalizeUpstreamModel(t *testing.T) {
	assert.Equal(t, "seedance-2-0", normalizeUpstreamModel("seedance-2.0 default"))
	assert.Equal(t, "seedance-2-0-fast", normalizeUpstreamModel("seedance-2.0-fast"))
	assert.Equal(t, "seedance-2-0-mini", normalizeUpstreamModel("seedance-2-0-mini"))
	assert.Equal(t, "seedance-2-0", normalizeUpstreamModel(""))
}

func TestNormalizeHolyCrabBaseURL(t *testing.T) {
	assert.Equal(t, "https://generate.holycrab.ai", normalizeHolyCrabBaseURL("https://generate.holycrab.ai/"))
	assert.Equal(t, "https://generate.holycrab.ai", normalizeHolyCrabBaseURL("https://generate.holycrab.ai/api/tasks/generation"))
	assert.Equal(t, "https://generate.holycrab.ai", normalizeHolyCrabBaseURL("https://generate.holycrab.ai/api/tasks"))
}
