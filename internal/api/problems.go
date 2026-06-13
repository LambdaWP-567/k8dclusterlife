package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/lambdawp-567/k8dclusterlife/internal/cluster"
)

type ProblemSource interface {
	GetProblems(ctx context.Context) ([]cluster.Problem, error)
}

func HandleProblems(src ProblemSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		problems, err := src.GetProblems(r.Context())
		if err != nil {
			http.Error(w, "failed to get problems", http.StatusInternalServerError)
			return
		}
		if problems == nil {
			problems = []cluster.Problem{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(problems)
	}
}
