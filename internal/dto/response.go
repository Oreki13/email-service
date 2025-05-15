package dto

const (
	StatusSuccess = "SUCCESS"
	StatusError   = "ERROR"
)

// APIResponse adalah format response standar untuk seluruh endpoint API
// Data dapat berupa objek apapun (interface{})
type APIResponse struct {
	Status  string      `json:"status"`
	TraceID string      `json:"trace_id"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
