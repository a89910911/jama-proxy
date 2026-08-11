package relay

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	taskdoubao "github.com/QuantumNous/new-api/relay/channel/task/doubao"
	tasksora "github.com/QuantumNous/new-api/relay/channel/task/sora"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTaskAdaptor_SeedanceOnOpenAIUsesDoubao(t *testing.T) {
	adaptor, platform := ResolveTaskAdaptor(constant.ChannelTypeOpenAI, "doubao-seedance-2-0-260128")
	require.NotNil(t, adaptor)
	_, ok := adaptor.(*taskdoubao.TaskAdaptor)
	assert.True(t, ok)
	assert.Equal(t, constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo)), platform)
}

func TestResolveTaskAdaptor_SeedanceOnVolcEngineUsesDoubao(t *testing.T) {
	adaptor, platform := ResolveTaskAdaptor(constant.ChannelTypeVolcEngine, "doubao-seedance-2-0-fast-260128")
	require.NotNil(t, adaptor)
	_, ok := adaptor.(*taskdoubao.TaskAdaptor)
	assert.True(t, ok)
	assert.Equal(t, constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo)), platform)
}

func TestResolveTaskAdaptor_SoraOnOpenAIStaysSora(t *testing.T) {
	adaptor, platform := ResolveTaskAdaptor(constant.ChannelTypeOpenAI, "sora-2")
	require.NotNil(t, adaptor)
	_, ok := adaptor.(*tasksora.TaskAdaptor)
	assert.True(t, ok)
	assert.Equal(t, constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeOpenAI)), platform)
}

func TestIsDoubaoSeedanceModel(t *testing.T) {
	assert.True(t, common.IsDoubaoSeedanceModel("doubao-seedance-2-0-260128"))
	assert.True(t, common.IsDoubaoSeedanceModel("seedance-1-0-pro-250528"))
	assert.False(t, common.IsDoubaoSeedanceModel("seedance-2-0"))
	assert.False(t, common.IsDoubaoSeedanceModel("seedance-2.0 default"))
	assert.False(t, common.IsDoubaoSeedanceModel("doubao-seed-2-1-pro-260628"))
	assert.False(t, common.IsDoubaoSeedanceModel("sora-2"))
}

func TestIsHolyCrabSeedanceModel(t *testing.T) {
	assert.True(t, common.IsHolyCrabSeedanceModel("seedance-2-0"))
	assert.True(t, common.IsHolyCrabSeedanceModel("seedance-2.0 default"))
	assert.True(t, common.IsHolyCrabSeedanceModel("seedance-2-0-fast"))
	assert.False(t, common.IsHolyCrabSeedanceModel("doubao-seedance-2-0-260128"))
	assert.False(t, common.IsHolyCrabSeedanceModel("seedance-1-0-pro-250528"))
}

func TestGetEndpointTypesSeedanceOnOpenAI(t *testing.T) {
	types := common.GetEndpointTypesByChannelType(constant.ChannelTypeOpenAI, "doubao-seedance-2-0-260128")
	assert.Equal(t, []constant.EndpointType{
		constant.EndpointTypeOpenAIVideo,
		constant.EndpointTypeOpenAI,
	}, types)
}

func TestGetEndpointTypesHolyCrabOnCustom(t *testing.T) {
	types := common.GetEndpointTypesByChannelType(constant.ChannelTypeCustom, "seedance-2.0 default")
	assert.Equal(t, []constant.EndpointType{
		constant.EndpointTypeOpenAIVideo,
		constant.EndpointTypeOpenAI,
	}, types)
}
