package relay

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTaskAdaptor_HolyCrab(t *testing.T) {
	adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeHolyCrab)))
	require.NotNil(t, adaptor)
	assert.Equal(t, "holycrab", adaptor.GetChannelName())
	assert.Contains(t, adaptor.GetModelList(), "seedance-2-0")
}

func TestResolveTaskAdaptor_HolyCrabSeedanceNotDoubao(t *testing.T) {
	adaptor, platform := ResolveTaskAdaptor(constant.ChannelTypeHolyCrab, "seedance-2-0")
	require.NotNil(t, adaptor)
	assert.Equal(t, "holycrab", adaptor.GetChannelName())
	assert.Equal(t, constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeHolyCrab)), platform)
}

func TestResolveTaskAdaptor_HolyCrabOnCustomUsesHolyCrab(t *testing.T) {
	adaptor, platform := ResolveTaskAdaptor(constant.ChannelTypeCustom, "seedance-2.0 default")
	require.NotNil(t, adaptor)
	assert.Equal(t, "holycrab", adaptor.GetChannelName())
	assert.Equal(t, constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeHolyCrab)), platform)
}

func TestResolveTaskAdaptor_HolyCrabOnOpenAIUsesHolyCrab(t *testing.T) {
	adaptor, platform := ResolveTaskAdaptor(constant.ChannelTypeOpenAI, "seedance-2-0-fast")
	require.NotNil(t, adaptor)
	assert.Equal(t, "holycrab", adaptor.GetChannelName())
	assert.Equal(t, constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeHolyCrab)), platform)
}
