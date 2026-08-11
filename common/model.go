package common

import "strings"

var (
	// OpenAIResponseOnlyModels is a list of models that are only available for OpenAI responses.
	OpenAIResponseOnlyModels = []string{
		"o3-pro",
		"o3-deep-research",
		"o4-mini-deep-research",
	}
	ImageGenerationModels = []string{
		"dall-e-3",
		"dall-e-2",
		"gpt-image-1",
		"prefix:imagen-",
		"flux-",
		"flux.1-",
	}
	// DoubaoSeedanceModels matches Volcengine/Ark Seedance video models.
	// These use the doubao contents/generations task API, not OpenAI Sora.
	DoubaoSeedanceModels = []string{
		"seedance",
	}
	// HolyCrabSeedanceModels matches HolyCrab Seedance 2.0 family model ids
	// (seedance-2-0 / seedance-2.0 and fast/mini variants).
	HolyCrabSeedanceModels = []string{
		"seedance-2-0",
		"seedance-2.0",
	}
	OpenAITextModels = []string{
		"gpt-",
		"o1",
		"o3",
		"o4",
		"chatgpt",
	}
)

func IsOpenAIResponseOnlyModel(modelName string) bool {
	for _, m := range OpenAIResponseOnlyModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

func IsImageGenerationModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range ImageGenerationModels {
		if strings.Contains(modelName, m) {
			return true
		}
		if strings.HasPrefix(m, "prefix:") && strings.HasPrefix(modelName, strings.TrimPrefix(m, "prefix:")) {
			return true
		}
	}
	return false
}

// IsHolyCrabSeedanceModel reports whether the model is a HolyCrab Seedance 2.0
// family id (including dotted forms like "seedance-2.0 default").
func IsHolyCrabSeedanceModel(modelName string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	if modelName == "" || strings.HasPrefix(modelName, "doubao-") {
		return false
	}
	for _, m := range HolyCrabSeedanceModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

// IsDoubaoSeedanceModel reports whether the model is a Doubao/Ark Seedance
// video model that must use the doubao task adaptor (not OpenAI Sora).
// HolyCrab Seedance 2.0 ids are excluded so they route to the HolyCrab adaptor.
func IsDoubaoSeedanceModel(modelName string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	if modelName == "" || IsHolyCrabSeedanceModel(modelName) {
		return false
	}
	for _, m := range DoubaoSeedanceModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

// IsSeedanceVideoModel reports whether the model should use the async video
// generation path (HolyCrab or Doubao Seedance).
func IsSeedanceVideoModel(modelName string) bool {
	return IsHolyCrabSeedanceModel(modelName) || IsDoubaoSeedanceModel(modelName)
}

func IsOpenAITextModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range OpenAITextModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}
