package holycrab

import "encoding/json"

type apiResult struct {
	Code      int             `json:"code"`
	Data      json.RawMessage `json:"data"`
	Message   string          `json:"message"`
	ErrorCode string          `json:"errorCode"`
	RequestID string          `json:"requestId"`
}

type generationRequest struct {
	Prompt             string   `json:"prompt"`
	Duration           int      `json:"duration"`
	Model              string   `json:"model"`
	Resolution         string   `json:"resolution"`
	Ratio              string   `json:"ratio,omitempty"`
	GenerateAudio      *bool    `json:"generate_audio,omitempty"`
	VideoAssetIds      []string `json:"videoAssetIds,omitempty"`
	ImageAssetIds      []string `json:"imageAssetIds,omitempty"`
	AudioAssetIds      []string `json:"audioAssetIds,omitempty"`
	FirstFrameAssetId  string   `json:"firstFrameAssetId,omitempty"`
	LastFrameAssetId   string   `json:"lastFrameAssetId,omitempty"`
}

type TaskVO struct {
	UniqID        string          `json:"uniqId"`
	Step          json.Number     `json:"step"`
	TaskType      string          `json:"taskType,omitempty"`
	Prompt        string          `json:"prompt,omitempty"`
	Model         string          `json:"model,omitempty"`
	AspectRatio   string          `json:"aspectRatio,omitempty"`
	VideoLength   int             `json:"videoLength,omitempty"`
	Resolution    string          `json:"resolution,omitempty"`
	GenerateAudio bool            `json:"generateAudio,omitempty"`
	FrozenCredit  int             `json:"frozenCredit,omitempty"`
	VideoURL      string          `json:"videoUrl,omitempty"`
	ImageURLs     json.RawMessage `json:"imageUrls,omitempty"`
	AudioIDs      json.RawMessage `json:"audioIds,omitempty"`
	Error         string          `json:"error,omitempty"`
	CreateTime    string          `json:"createTime,omitempty"`
	UpdateTime    string          `json:"updateTime,omitempty"`
}
