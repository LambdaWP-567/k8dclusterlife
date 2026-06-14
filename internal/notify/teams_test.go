package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lambdawp-567/k8dclusterlife/internal/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotifyProblem_SendsRequest(t *testing.T) {
	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		received = buf
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := &Notifier{webhookURL: srv.URL, client: srv.Client()}
	n.NotifyProblem(cluster.Problem{
		Kind:        "Pod",
		Name:        "crash-pod",
		Namespace:   "default",
		ClusterName: "prod",
		Status:      "CrashLoopBackOff",
		Description: "Container crasht",
		Cause:       "Fehler im Startprozess",
		Severity:    cluster.SeverityCritical,
	})

	require.NotEmpty(t, received)
	var payload map[string]any
	err := json.Unmarshal(received, &payload)
	require.NoError(t, err)
	assert.Equal(t, "message", payload["type"])
}

func TestNotifyProblem_NoWebhook(t *testing.T) {
	// Should not panic when webhook URL is empty
	n := &Notifier{webhookURL: "", client: http.DefaultClient}
	n.NotifyProblem(cluster.Problem{Kind: "Pod", Name: "test"})
}

func TestAdaptiveCard_Structure(t *testing.T) {
	card := adaptiveCard("Test Title", []cardFact{
		{Title: "Key", Value: "Value"},
		{Title: "Empty", Value: ""},
	})
	assert.Equal(t, "message", card["type"])
	attachments, ok := card["attachments"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, attachments, 1)
}
