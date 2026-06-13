package testorch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAll_ReturnsScenarios(t *testing.T) {
	scenarios := All()
	assert.GreaterOrEqual(t, len(scenarios), 5)
	for _, s := range scenarios {
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.Description)
		assert.NotNil(t, s.Setup)
		assert.NotNil(t, s.Teardown)
	}
}

func TestGet_Found(t *testing.T) {
	s, err := Get("crashloop")
	require.NoError(t, err)
	assert.Equal(t, "crashloop", s.Name)
}

func TestGet_NotFound(t *testing.T) {
	_, err := Get("nonexistent")
	assert.Error(t, err)
}

func TestScenarioNames(t *testing.T) {
	expected := []string{"crashloop", "imagepullbackoff", "scale-zero", "pending", "node-cordon"}
	scenarios := All()
	names := make([]string, len(scenarios))
	for i, s := range scenarios {
		names[i] = s.Name
	}
	for _, name := range expected {
		assert.Contains(t, names, name)
	}
}
