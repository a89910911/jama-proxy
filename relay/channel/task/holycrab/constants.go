package holycrab

const ChannelName = "holycrab"

const (
	GenerationEndpoint = "/api/tasks/generation"
	TaskDetailEndpoint = "/api/tasks"
)

const (
	MinDuration       = 4
	MaxDuration       = 15
	DefaultDuration   = 5
	DefaultResolution = "720p"
)

const (
	TaskStepCreated    = 0
	TaskStepGenerating = 1
	TaskStepCompleted  = 2
	TaskStepFailed     = 3
)

const (
	APIResultSuccess = 200
)

var ModelList = []string{
	"seedance-2-0",
	"seedance-2-0-fast",
	"seedance-2-0-mini",
}
