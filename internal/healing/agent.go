package healing

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/lambdawp-567/k8dclusterlife/internal/cluster"
)

const systemPrompt = `Du bist ein Kubernetes-Heilungsagent. Deine Aufgabe ist es, Probleme in einem Kubernetes-Cluster zu diagnostizieren und zu beheben.

Vorgehen:
1. Sammle zuerst alle relevanten Informationen mit Lese-Tools (get_pod_logs, get_events, describe_resource, get_node_status)
2. Analysiere die Ursache des Problems
3. Führe dann gezielt Heilungsmaßnahmen durch (delete_pod, scale_deployment, etc.)
4. Prüfe nach jeder Aktion ob das Problem behoben wurde
5. Falls eine Aktion die Lage verschlechtert, erkläre das klar

Wichtige Regeln:
- Erkläre jeden Schritt auf Deutsch für einen nicht-technischen Benutzer
- Führe schreibende Aktionen nur aus wenn nötig
- Fasse am Ende zusammen ob das Problem gelöst wurde und warum`

// Agent runs the Claude agentic healing loop.
type Agent struct {
	client   anthropic.Client
	model    string
	executor *Executor

	mu       sync.Mutex
	sessions map[string]*sessionState
}

type sessionState struct {
	session  *Session
	approval chan approvalResult
	done     chan struct{}
}

type approvalResult struct {
	approved bool
}

func New(executor *Executor) *Agent {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	model := os.Getenv("CLAUDE_MODEL")
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &Agent{
		client:   client,
		model:    model,
		executor: executor,
		sessions: make(map[string]*sessionState),
	}
}

// StartSession initiates a healing session and returns the session ID.
func (a *Agent) StartSession(
	ctx context.Context,
	problem cluster.Problem,
	mode AutonomyMode,
	countdownSec int,
	events chan<- StreamEvent,
) *Session {
	session := &Session{
		ID:           fmt.Sprintf("heal-%d", time.Now().UnixNano()),
		ClusterID:    problem.ClusterID,
		ProblemID:    problem.ID,
		ProblemKind:  problem.Kind,
		ProblemName:  problem.Name,
		ProblemNS:    problem.Namespace,
		Status:       StatusRunning,
		AutonomyMode: mode,
		StartedAt:    time.Now(),
	}

	state := &sessionState{
		session:  session,
		done:     make(chan struct{}),
		approval: make(chan approvalResult, 1),
	}

	a.mu.Lock()
	a.sessions[session.ID] = state
	a.mu.Unlock()

	go a.run(ctx, state, problem, mode, countdownSec, events)
	return session
}

// ApproveAction approves or rejects a pending write tool execution.
func (a *Agent) ApproveAction(sessionID string, approved bool) error {
	a.mu.Lock()
	state, ok := a.sessions[sessionID]
	a.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}
	select {
	case state.approval <- approvalResult{approved: approved}:
	default:
		return fmt.Errorf("no pending approval")
	}
	return nil
}

// GetSession returns the current session state.
func (a *Agent) GetSession(sessionID string) *Session {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.sessions[sessionID]; ok {
		return s.session
	}
	return nil
}

func (a *Agent) run(
	ctx context.Context,
	state *sessionState,
	problem cluster.Problem,
	mode AutonomyMode,
	countdownSec int,
	events chan<- StreamEvent,
) {
	defer close(state.done)

	emit := func(ev StreamEvent) {
		select {
		case events <- ev:
		default:
		}
	}

	userMsg := fmt.Sprintf(`Problem erkannt:
Kind: %s
Name: %s
Namespace: %s
Status: %s
Beschreibung: %s
Ursache: %s

Bitte diagnostiziere und behebe dieses Problem.`,
		problem.Kind, problem.Name, problem.Namespace,
		problem.Status, problem.Description, problem.Cause)

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)),
	}

	a.mu.Lock()
	state.session.Dialog = append(state.session.Dialog, Message{
		Role:    "user",
		Content: userMsg,
		At:      time.Now(),
	})
	a.mu.Unlock()

	tools := toolDefs()

	for i := 0; i < 20; i++ {
		resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(a.model),
			MaxTokens: 4096,
			System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
			Messages:  messages,
			Tools:     tools,
		})
		if err != nil {
			slog.Error("claude api error", "error", err, "session", state.session.ID)
			emit(StreamEvent{Type: "error", Content: "Claude API Fehler: " + err.Error()})
			a.finalizeSession(state, StatusFailed, "Claude API Fehler: "+err.Error())
			return
		}

		// Process response content blocks
		var toolUses []anthropic.ToolUseBlock
		var paramBlocks []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			paramBlocks = append(paramBlocks, block.ToParam())
			switch b := block.AsAny().(type) {
			case anthropic.TextBlock:
				emit(StreamEvent{Type: "text", Content: b.Text})
				a.mu.Lock()
				state.session.Dialog = append(state.session.Dialog, Message{
					Role:    "assistant",
					Content: b.Text,
					At:      time.Now(),
				})
				a.mu.Unlock()
			case anthropic.ToolUseBlock:
				toolUses = append(toolUses, b)
			}
		}

		messages = append(messages, anthropic.NewAssistantMessage(paramBlocks...))

		if len(toolUses) == 0 || resp.StopReason == "end_turn" {
			a.finalizeSession(state, StatusHealed, "Heilung abgeschlossen.")
			emit(StreamEvent{Type: "done", Content: "Heilung abgeschlossen."})
			return
		}

		// Execute tool calls
		var toolResults []anthropic.ContentBlockParamUnion
		for _, tu := range toolUses {
			inputStr := string(tu.Input)
			emit(StreamEvent{Type: "tool_call", Tool: tu.Name, Input: inputStr})

			var output string
			var toolErr error

			if IsWriteTool(tu.Name) {
				switch mode {
				case ModeManual:
					actionID := fmt.Sprintf("%s-%d", tu.Name, time.Now().UnixNano())
					emit(StreamEvent{
						Type:     "approval_required",
						Tool:     tu.Name,
						Input:    inputStr,
						ActionID: actionID,
					})
					select {
					case result := <-state.approval:
						if !result.approved {
							output = "Aktion vom Benutzer abgelehnt."
						} else {
							output, toolErr = a.executor.Execute(ctx, tu.Name, inputStr)
						}
					case <-ctx.Done():
						a.finalizeSession(state, StatusAborted, "Abgebrochen.")
						emit(StreamEvent{Type: "done", Content: "Abgebrochen."})
						return
					}

				case ModeCountdown:
					actionID := fmt.Sprintf("%s-%d", tu.Name, time.Now().UnixNano())
					emit(StreamEvent{
						Type:     "approval_required",
						Tool:     tu.Name,
						Input:    inputStr,
						ActionID: actionID,
					})
					timer := time.NewTimer(time.Duration(countdownSec) * time.Second)
					select {
					case result := <-state.approval:
						timer.Stop()
						if !result.approved {
							output = "Aktion vom Benutzer abgelehnt."
						} else {
							output, toolErr = a.executor.Execute(ctx, tu.Name, inputStr)
						}
					case <-timer.C:
						output, toolErr = a.executor.Execute(ctx, tu.Name, inputStr)
					case <-ctx.Done():
						timer.Stop()
						a.finalizeSession(state, StatusAborted, "Abgebrochen.")
						return
					}

				default: // FULL_AUTO
					output, toolErr = a.executor.Execute(ctx, tu.Name, inputStr)
				}
			} else {
				output, toolErr = a.executor.Execute(ctx, tu.Name, inputStr)
			}

			if toolErr != nil {
				output = "Fehler: " + toolErr.Error()
			}

			emit(StreamEvent{Type: "tool_result", Tool: tu.Name, Output: output})

			a.mu.Lock()
			state.session.Actions = append(state.session.Actions, Action{
				ID:         tu.ID,
				ToolName:   tu.Name,
				ToolInput:  inputStr,
				Output:     output,
				ExecutedAt: time.Now(),
			})
			a.mu.Unlock()

			toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, output, toolErr != nil))
		}

		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}

	a.finalizeSession(state, StatusFailed, "Maximale Iterationen erreicht.")
	emit(StreamEvent{Type: "done", Content: "Maximale Iterationen erreicht — Problem möglicherweise nicht vollständig gelöst."})
}

func (a *Agent) finalizeSession(state *sessionState, status SessionStatus, result string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	state.session.Status = status
	state.session.Result = result
	state.session.FinishedAt = &now
}
