package health

// Status represents the outcome of a health check.
type Status string

const (
	StatusOK       Status = "ok"
	StatusWarning  Status = "warning"
	StatusFailed   Status = "failed"
	StatusDegraded Status = "degraded"
)

// CheckResult captures the outcome of a health check.
type CheckResult struct {
	Name      string                 `json:"name"`
	Component string                 `json:"component"`
	Status    Status                 `json:"status"`
	Message   string                 `json:"message"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}
