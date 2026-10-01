/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package extgatlingenterprise

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-gatling/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGatlingEnterpriseRunAction(t *testing.T) {
	action := NewGatlingEnterpriseRunAction()

	require.NotNil(t, action)
	assert.Equal(t, RunState{}, action.NewEmptyState())
}

func TestDescribe(t *testing.T) {
	description := RunAction{}.Describe()

	assert.Equal(t, actionId, description.Id)
	assert.Equal(t, "Gatling Enterprise", description.Label)
	assert.Equal(t, action_kit_api.LoadTest, description.Kind)
	assert.Equal(t, action_kit_api.TimeControlInternal, description.TimeControl)
	require.NotNil(t, description.TargetSelection)
	assert.Equal(t, targetType, description.TargetSelection.TargetType)
	assert.Equal(t, new(action_kit_api.QuantityRestrictionExactlyOne), description.TargetSelection.QuantityRestriction)

	names := make([]string, 0, len(description.Parameters))
	for _, p := range description.Parameters {
		names = append(names, p.Name)
	}
	assert.Equal(t, []string{"duration", "systemProperties", "environmentVariables"}, names)
	assert.NotNil(t, description.Status)
	assert.NotNil(t, description.Stop)
}

func TestStart_StartsTheSimulation(t *testing.T) {
	var started GatlingStartSimulationRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/simulations/start", r.URL.Path)
		assert.Equal(t, "sim-1", r.URL.Query().Get("simulation"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&started))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GatlingStartResponse{RunId: "run-1"})
	}))
	t.Cleanup(server.Close)
	originalConfig := config.Config
	t.Cleanup(func() { config.Config = originalConfig })
	config.Config.EnterpriseApiBaseUrl = server.URL
	config.Config.EnterpriseApiToken = "test-token"

	state := RunState{
		SimulationId:         "sim-1",
		ExperimentKey:        "EXP-1",
		ExecutionId:          42,
		SystemProperties:     map[string]string{"users": "10"},
		EnvironmentVariables: map[string]string{"STAGE": "dev"},
	}
	result, err := RunAction{}.Start(context.Background(), &state)

	require.NoError(t, err)
	assert.Equal(t, "run-1", state.RunId)
	assert.Equal(t, "Steadybit - EXP-1 - 42", started.Title)
	assert.Equal(t, "Executed by Steadybit Experiment EXP-1, Execution 42", started.Description)
	assert.Equal(t, map[string]string{"users": "10"}, started.ExtraSystemProperties)
	assert.Equal(t, map[string]string{"STAGE": "dev"}, started.ExtraEnvironmentVariables)
	// Links into the Gatling Enterprise UI are only known for the default (cloud) api.
	require.NotNil(t, result.Messages)
	assert.Empty(t, *result.Messages)
}
