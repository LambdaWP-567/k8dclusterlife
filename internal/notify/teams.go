package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/lambdawp-567/k8dclusterlife/internal/cluster"
	"github.com/lambdawp-567/k8dclusterlife/internal/healing"
)

// Notifier sends Teams webhook notifications.
type Notifier struct {
	webhookURL string
	client     *http.Client
}

func NewNotifier() *Notifier {
	return &Notifier{
		webhookURL: os.Getenv("TEAMS_WEBHOOK_URL"),
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// NotifyProblem sends an Adaptive Card for a new problem.
func (n *Notifier) NotifyProblem(p cluster.Problem) {
	if n.webhookURL == "" {
		return
	}

	severityEmoji := map[cluster.Severity]string{
		cluster.SeverityCritical: "🔴",
		cluster.SeverityWarning:  "🟡",
		cluster.SeverityInfo:     "🔵",
	}[p.Severity]

	card := adaptiveCard(
		fmt.Sprintf("%s Problem erkannt: %s/%s", severityEmoji, p.Kind, p.Name),
		[]cardFact{
			{Title: "Cluster", Value: p.ClusterName},
			{Title: "Namespace", Value: p.Namespace},
			{Title: "Status", Value: p.Status},
			{Title: "Beschreibung", Value: p.Description},
			{Title: "Ursache", Value: p.Cause},
		},
	)
	n.send(card)
}

// NotifyHealed sends a notification when a problem is healed.
func (n *Notifier) NotifyHealed(sess *healing.Session) {
	if n.webhookURL == "" {
		return
	}
	card := adaptiveCard(
		fmt.Sprintf("✅ Geheilt: %s/%s", sess.ProblemKind, sess.ProblemName),
		[]cardFact{
			{Title: "Cluster", Value: sess.ClusterID},
			{Title: "Ergebnis", Value: sess.Result},
		},
	)
	n.send(card)
}

func (n *Notifier) send(payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("teams: marshal failed", "error", err)
		return
	}
	resp, err := n.client.Post(n.webhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		slog.Error("teams: webhook failed", "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		slog.Warn("teams: unexpected status", "status", resp.StatusCode)
	}
}

// --- Adaptive Card helpers ---

type cardFact struct {
	Title string
	Value string
}

func adaptiveCard(title string, facts []cardFact) map[string]any {
	factItems := make([]map[string]any, 0, len(facts))
	for _, f := range facts {
		if f.Value == "" {
			continue
		}
		factItems = append(factItems, map[string]any{
			"title": f.Title,
			"value": f.Value,
		})
	}

	return map[string]any{
		"type": "message",
		"attachments": []map[string]any{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]any{
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"type":    "AdaptiveCard",
					"version": "1.4",
					"body": []map[string]any{
						{
							"type":   "TextBlock",
							"text":   title,
							"weight": "bolder",
							"size":   "medium",
							"wrap":   true,
						},
						{
							"type":  "FactSet",
							"facts": factItems,
						},
					},
				},
			},
		},
	}
}
