package healing

import "time"

type AutonomyMode string

const (
	ModeManual   AutonomyMode = "MANUAL"
	ModeCountdown AutonomyMode = "COUNTDOWN"
	ModeFullAuto  AutonomyMode = "FULL_AUTO"
)

type SessionStatus string

const (
	StatusRunning  SessionStatus = "running"
	StatusHealed   SessionStatus = "healed"
	StatusFailed   SessionStatus = "failed"
	StatusAborted  SessionStatus = "aborted"
)

// Session tracks one healing attempt.
type Session struct {
	ID           string        `json:"id"`
	ClusterID    string        `json:"cluster_id"`
	ProblemID    string        `json:"problem_id"`
	ProblemKind  string        `json:"problem_kind"`
	ProblemName  string        `json:"problem_name"`
	ProblemNS    string        `json:"problem_namespace"`
	Status       SessionStatus `json:"status"`
	AutonomyMode AutonomyMode  `json:"autonomy_mode"`
	Result       string        `json:"result,omitempty"`
	StartedAt    time.Time     `json:"started_at"`
	FinishedAt   *time.Time    `json:"finished_at,omitempty"`
	Dialog       []Message     `json:"dialog"`
	Actions      []Action      `json:"actions"`
}

// Message is a single turn in the Claude conversation.
type Message struct {
	Role    string `json:"role"` // "assistant" | "user" | "tool_result"
	Content string `json:"content"`
	At      time.Time `json:"at"`
}

// Action is a write tool call that requires approval in MANUAL/COUNTDOWN modes.
type Action struct {
	ID        string    `json:"id"`
	ToolName  string    `json:"tool_name"`
	ToolInput string    `json:"tool_input"` // JSON
	Approved  *bool     `json:"approved,omitempty"`
	Output    string    `json:"output,omitempty"`
	ExecutedAt time.Time `json:"executed_at"`
}

// StreamEvent is sent over WebSocket during a healing session.
type StreamEvent struct {
	Type    string `json:"type"` // "text" | "tool_call" | "tool_result" | "done" | "error" | "approval_required"
	Content string `json:"content,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Input   string `json:"input,omitempty"`
	Output  string `json:"output,omitempty"`
	ActionID string `json:"action_id,omitempty"`
}
