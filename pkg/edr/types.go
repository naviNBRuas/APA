package edr

import "time"

// ResponseAction defines an automated response action
type ResponseAction struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ActionType  string    `json:"action_type"` // quarantine, terminate, isolate, self-destruct
	Severity    string    `json:"severity"`    // low, medium, high, critical
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}
