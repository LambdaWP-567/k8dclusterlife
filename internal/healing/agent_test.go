package healing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsWriteTool(t *testing.T) {
	assert.True(t, IsWriteTool("delete_pod"))
	assert.True(t, IsWriteTool("scale_deployment"))
	assert.True(t, IsWriteTool("cordon_node"))
	assert.True(t, IsWriteTool("uncordon_node"))

	assert.False(t, IsWriteTool("get_pod_logs"))
	assert.False(t, IsWriteTool("get_events"))
	assert.False(t, IsWriteTool("describe_resource"))
	assert.False(t, IsWriteTool("get_node_status"))
}

func TestToolDefs(t *testing.T) {
	tools := toolDefs()
	// Verify all expected tools are defined
	names := make(map[string]bool)
	for _, t := range tools {
		if t.OfTool != nil {
			names[t.OfTool.Name] = true
		}
	}
	expected := []string{
		"get_pod_logs", "get_events", "describe_resource", "get_node_status",
		"delete_pod", "scale_deployment", "cordon_node", "uncordon_node",
	}
	for _, name := range expected {
		assert.True(t, names[name], "tool %s should be defined", name)
	}
}

func TestApproveAction_SessionNotFound(t *testing.T) {
	a := &Agent{sessions: make(map[string]*sessionState)}
	err := a.ApproveAction("nonexistent", true)
	assert.Error(t, err)
}

func TestGetSession_NotFound(t *testing.T) {
	a := &Agent{sessions: make(map[string]*sessionState)}
	session := a.GetSession("nonexistent")
	assert.Nil(t, session)
}

func TestFinalizeSession(t *testing.T) {
	a := &Agent{sessions: make(map[string]*sessionState)}
	state := &sessionState{session: &Session{Status: StatusRunning}}
	a.finalizeSession(state, StatusHealed, "done")

	assert.Equal(t, StatusHealed, state.session.Status)
	assert.Equal(t, "done", state.session.Result)
	assert.NotNil(t, state.session.FinishedAt)
}
