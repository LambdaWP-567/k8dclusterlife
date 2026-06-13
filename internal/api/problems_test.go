package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lambdawp-567/k8dclusterlife/internal/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProblemSource struct {
	problems []cluster.Problem
	err      error
}

func (m *mockProblemSource) GetProblems(_ context.Context) ([]cluster.Problem, error) {
	return m.problems, m.err
}

func TestHandleProblems_Empty(t *testing.T) {
	h := HandleProblems(&mockProblemSource{})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/api/problems", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	var out []cluster.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.Empty(t, out)
}

func TestHandleProblems_WithProblems(t *testing.T) {
	src := &mockProblemSource{
		problems: []cluster.Problem{
			{ID: "1", Name: "nginx", Status: "CrashLoopBackOff"},
		},
	}
	h := HandleProblems(src)
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/api/problems", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	var out []cluster.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.Len(t, out, 1)
	assert.Equal(t, "nginx", out[0].Name)
}
