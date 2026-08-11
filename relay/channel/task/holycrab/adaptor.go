package holycrab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = normalizeHolyCrabBaseURL(info.ChannelBaseUrl)
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *taskdto.TaskError {
	if err := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); err != nil {
		return err
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapper(err, "get_task_request_failed", http.StatusBadRequest)
	}
	duration := taskDuration(req)
	if duration < MinDuration || duration > MaxDuration {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("duration must be between %d and %d seconds", MinDuration, MaxDuration),
			"invalid_duration",
			http.StatusBadRequest,
		)
	}
	return nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	seconds := taskDuration(req)
	if seconds > relaycommon.MaxTaskDurationSeconds {
		seconds = relaycommon.MaxTaskDurationSeconds
	}
	return map[string]float64{"seconds": float64(seconds)}
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return a.baseURL + GenerationEndpoint, nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-User-Token", a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req, ok := v.(relaycommon.TaskSubmitReq)
	if !ok {
		return nil, fmt.Errorf("invalid request type in context")
	}

	body, err := a.convertToRequestPayload(&req, info)
	if err != nil {
		return nil, errors.Wrap(err, "convert request payload failed")
	}

	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *taskdto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	taskVO, apiErr := parseAPIResultTask(responseBody)
	if apiErr != nil {
		taskErr = apiErr
		return
	}
	if taskVO.UniqID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("uniqId is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return taskVO.UniqID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}

	baseUrl = normalizeHolyCrabBaseURL(baseUrl)
	url := fmt.Sprintf("%s%s/%s", baseUrl, TaskDetailEndpoint, taskID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-User-Token", key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) (*generationRequest, error) {
	payload := &generationRequest{
		Prompt:     req.Prompt,
		Duration:   taskDuration(*req),
		Model:      normalizeUpstreamModel(info.UpstreamModelName),
		Resolution: parseResolution(req.Size),
		Ratio:      parseRatio(req.Size),
	}
	if err := taskcommon.UnmarshalMetadata(req.Metadata, payload); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}
	if payload.Resolution == "" {
		payload.Resolution = DefaultResolution
	}
	payload.Model = normalizeUpstreamModel(payload.Model)
	return payload, nil
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	taskVO, err := parseTaskVOResponse(respBody)
	if err != nil {
		return nil, err
	}
	return mapTaskVOToTaskInfo(taskVO), nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	taskVO, err := parseTaskVOResponse(originTask.Data)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal holycrab task data failed")
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = originTask.TaskID
	openAIVideo.Status = originTask.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(originTask.Progress)
	openAIVideo.CreatedAt = originTask.CreatedAt
	openAIVideo.CompletedAt = originTask.UpdatedAt
	openAIVideo.Model = originTask.Properties.OriginModelName

	videoURL := taskVO.VideoURL
	if videoURL == "" {
		videoURL = originTask.GetResultURL()
	}
	step, stepErr := parseTaskStep(taskVO.Step)
	if (stepErr == nil && step == TaskStepCompleted && videoURL != "") ||
		(originTask.Status == model.TaskStatusSuccess && videoURL != "") {
		openAIVideo.SetMetadata("url", videoURL)
	}
	if stepErr == nil && step == TaskStepFailed && taskVO.Error != "" {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: taskVO.Error,
			Code:    "task_failed",
		}
	}

	return common.Marshal(openAIVideo)
}

func parseAPIResultTask(respBody []byte) (*TaskVO, *taskdto.TaskError) {
	var wrapper apiResult
	if err := common.Unmarshal(respBody, &wrapper); err != nil {
		return nil, service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", respBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	if wrapper.Code != APIResultSuccess {
		message := wrapper.Message
		if message == "" {
			message = "holycrab api error"
		}
		code := wrapper.ErrorCode
		if code == "" {
			code = strconv.Itoa(wrapper.Code)
		}
		return nil, service.TaskErrorWrapper(fmt.Errorf("%s", message), code, http.StatusBadRequest)
	}
	if len(wrapper.Data) == 0 {
		return nil, service.TaskErrorWrapper(fmt.Errorf("empty response data"), "invalid_response", http.StatusInternalServerError)
	}

	var taskVO TaskVO
	if err := common.Unmarshal(wrapper.Data, &taskVO); err != nil {
		return nil, service.TaskErrorWrapper(errors.Wrap(err, "unmarshal task data failed"), "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	return &taskVO, nil
}

func parseTaskVOResponse(respBody []byte) (*TaskVO, error) {
	var wrapper apiResult
	if err := common.Unmarshal(respBody, &wrapper); err != nil {
		return nil, errors.Wrap(err, "unmarshal api result failed")
	}
	if wrapper.Code == APIResultSuccess && len(wrapper.Data) > 0 {
		var taskVO TaskVO
		if err := common.Unmarshal(wrapper.Data, &taskVO); err != nil {
			return nil, errors.Wrap(err, "unmarshal task vo failed")
		}
		return &taskVO, nil
	}

	var taskVO TaskVO
	if err := common.Unmarshal(respBody, &taskVO); err != nil {
		return nil, errors.Wrap(err, "unmarshal task vo failed")
	}
	if taskVO.UniqID != "" || taskVO.Step != "" {
		return &taskVO, nil
	}
	if wrapper.Message != "" {
		return nil, fmt.Errorf("holycrab api error: %s", wrapper.Message)
	}
	return nil, fmt.Errorf("unable to parse holycrab task response")
}

func mapTaskVOToTaskInfo(taskVO *TaskVO) *relaycommon.TaskInfo {
	taskInfo := &relaycommon.TaskInfo{
		TaskID: taskVO.UniqID,
	}
	step, err := parseTaskStep(taskVO.Step)
	if err != nil {
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = taskcommon.ProgressInProgress
		return taskInfo
	}

	switch step {
	case TaskStepCreated:
		taskInfo.Status = model.TaskStatusSubmitted
		taskInfo.Progress = taskcommon.ProgressSubmitted
	case TaskStepGenerating:
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = taskcommon.ProgressInProgress
	case TaskStepCompleted:
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = taskcommon.ProgressComplete
		taskInfo.Url = taskVO.VideoURL
	case TaskStepFailed:
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = taskcommon.ProgressComplete
		taskInfo.Reason = taskVO.Error
	default:
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = taskcommon.ProgressInProgress
	}
	return taskInfo
}

func parseTaskStep(step json.Number) (int, error) {
	if step == "" {
		return -1, fmt.Errorf("empty step")
	}
	value, err := step.Int64()
	if err != nil {
		return -1, err
	}
	return int(value), nil
}

func taskDuration(req relaycommon.TaskSubmitReq) int {
	duration := req.Duration
	if duration == 0 && req.Seconds != "" {
		duration, _ = strconv.Atoi(req.Seconds)
	}
	if duration == 0 {
		duration = DefaultDuration
	}
	return duration
}

func normalizeHolyCrabBaseURL(baseURL string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	normalized = strings.TrimSuffix(normalized, GenerationEndpoint)
	normalized = strings.TrimSuffix(normalized, TaskDetailEndpoint)
	return strings.TrimRight(normalized, "/")
}

func normalizeUpstreamModel(modelName string) string {
	normalized := strings.ToLower(strings.TrimSpace(modelName))
	normalized = strings.ReplaceAll(normalized, ".", "-")
	if i := strings.IndexAny(normalized, " \t"); i > 0 {
		normalized = normalized[:i]
	}
	switch {
	case strings.Contains(normalized, "seedance-2-0-mini"):
		return "seedance-2-0-mini"
	case strings.Contains(normalized, "seedance-2-0-fast"):
		return "seedance-2-0-fast"
	case strings.Contains(normalized, "seedance-2-0"):
		return "seedance-2-0"
	case normalized == "":
		return "seedance-2-0"
	default:
		return normalized
	}
}

func parseResolution(size string) string {
	normalized := strings.ToLower(strings.TrimSpace(size))
	switch normalized {
	case "", "default":
		return DefaultResolution
	case "480p", "720p", "1080p", "4k":
		return normalized
	case "3840x2160", "2160x3840":
		return "4k"
	case "1920x1080", "1080x1920":
		return "1080p"
	case "1280x720", "720x1280":
		return "720p"
	case "512x512", "854x480", "480x854":
		return "480p"
	}
	switch {
	case strings.Contains(normalized, "4k") || strings.Contains(normalized, "3840"):
		return "4k"
	case strings.Contains(normalized, "1080"):
		return "1080p"
	case strings.Contains(normalized, "768"):
		return "720p"
	case strings.Contains(normalized, "720"):
		return "720p"
	case strings.Contains(normalized, "512"), strings.Contains(normalized, "480"):
		return "480p"
	default:
		return DefaultResolution
	}
}

func parseRatio(size string) string {
	normalized := strings.TrimSpace(size)
	switch normalized {
	case "1280x720", "1920x1080", "3840x2160":
		return "16:9"
	case "720x1280", "1080x1920":
		return "9:16"
	case "1024x1024", "512x512":
		return "1:1"
	case "16:9", "4:3", "1:1", "3:4", "9:16", "21:9":
		return normalized
	default:
		return ""
	}
}
