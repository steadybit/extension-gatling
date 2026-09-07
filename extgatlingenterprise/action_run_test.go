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

func TestStatusToString(t *testing.T) {
	cases := map[int]string{
		0:  "Building",
		1:  "Deploying",
		2:  "Deployed",
		3:  "Injecting",
		4:  "Successful",
		5:  "AssertionsSuccessful",
		6:  "AutomaticallyStopped",
		7:  "Stopped",
		8:  "AssertionsFailed",
		9:  "Timeout",
		10: "BuildFailed",
		11: "Broken",
		12: "DeploymentFailed",
		13: "InsufficientLicense",
		15: "StopRequested",
		14: "Unknown (14)",
		99: "Unknown (99)",
	}
	for status, expected := range cases {
		assert.Equal(t, expected, statusToString(status), "status %d", status)
	}
}

func TestAppendDots(t *testing.T) {
	assert.Equal(t, "...", appendDots(0))
	assert.Equal(t, "...", appendDots(3))
	assert.Equal(t, "", appendDots(4))
	assert.Equal(t, "", appendDots(8))
}

func prepareRequest(simulationIds []string, cfg map[string]any) action_kit_api.PrepareActionRequestBody {
	return action_kit_api.PrepareActionRequestBody{
		Target: &action_kit_api.Target{
			Attributes: map[string][]string{"gatling.simulation.id": simulationIds},
		},
		Config: cfg,
		ExecutionContext: &action_kit_api.ExecutionContext{
			ExecutionId:   new(42),
			ExperimentKey: new("EXP-1"),
		},
	}
}

func TestPrepare(t *testing.T) {
	action := RunAction{}

	t.Run("fails without simulation id", func(t *testing.T) {
		state := RunState{}
		_, err := action.Prepare(context.Background(), &state, prepareRequest([]string{}, nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "No simulation id provided")
	})

	t.Run("fails with more than one simulation id", func(t *testing.T) {
		state := RunState{}
		_, err := action.Prepare(context.Background(), &state, prepareRequest([]string{"sim-1", "sim-2"}, nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "More than one simulation id provided")
	})

	t.Run("copies context and key/value config into the state", func(t *testing.T) {
		state := RunState{}
		cfg := map[string]any{
			"systemProperties":     []any{map[string]any{"key": "prop", "value": "p-value"}},
			"environmentVariables": []any{map[string]any{"key": "ENV", "value": "e-value"}},
		}
		result, err := action.Prepare(context.Background(), &state, prepareRequest([]string{"sim-1"}, cfg))
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "sim-1", state.SimulationId)
		assert.Equal(t, 42, state.ExecutionId)
		assert.Equal(t, "EXP-1", state.ExperimentKey)
		assert.Equal(t, map[string]string{"prop": "p-value"}, state.SystemProperties)
		assert.Equal(t, map[string]string{"ENV": "e-value"}, state.EnvironmentVariables)
		assert.Equal(t, -1, state.LastState)
	})

	t.Run("fails on malformed system properties", func(t *testing.T) {
		state := RunState{}
		cfg := map[string]any{"systemProperties": "not-a-list"}
		_, err := action.Prepare(context.Background(), &state, prepareRequest([]string{"sim-1"}, cfg))
		require.Error(t, err)
	})

	t.Run("fails on malformed environment variables", func(t *testing.T) {
		state := RunState{}
		cfg := map[string]any{"environmentVariables": "not-a-list"}
		_, err := action.Prepare(context.Background(), &state, prepareRequest([]string{"sim-1"}, cfg))
		require.Error(t, err)
	})
}

// withMockApi points the enterprise API config at a test server that answers
// GET /run with the given run response and records whether an abort was requested.
func withMockApi(t *testing.T, run *GatlingRunResponse) (aborted *bool) {
	t.Helper()
	aborted = new(false)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/run":
			if run == nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(run)
		case "/simulations/abort":
			*aborted = true
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)

	originalConfig := config.Config
	t.Cleanup(func() { config.Config = originalConfig })
	config.Config.EnterpriseApiBaseUrl = server.URL
	config.Config.EnterpriseApiToken = "test-token"
	return aborted
}

func TestStart_FailsWhenTheApiRejectsTheRun(t *testing.T) {
	withMockApi(t, nil)
	state := RunState{SimulationId: "sim-1", ExperimentKey: "EXP-1", ExecutionId: 1}
	_, err := RunAction{}.Start(context.Background(), &state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to run simulation")
}

func TestStatus(t *testing.T) {
	action := RunAction{}

	t.Run("fails when the run cannot be fetched", func(t *testing.T) {
		withMockApi(t, nil)
		state := RunState{RunId: "run-1", LastState: -1}
		_, err := action.Status(context.Background(), &state)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Failed to get run info")
	})

	t.Run("reports a state change while running", func(t *testing.T) {
		withMockApi(t, &GatlingRunResponse{Status: 3})
		state := RunState{RunId: "run-1", LastState: -1}
		result, err := action.Status(context.Background(), &state)
		require.NoError(t, err)
		assert.False(t, result.Completed)
		assert.Nil(t, result.Error)
		assert.Equal(t, 3, state.LastState)
		require.NotNil(t, result.Messages)
		assert.Equal(t, "- Injecting...", (*result.Messages)[1].Message)
	})

	t.Run("is silent when the state did not change", func(t *testing.T) {
		withMockApi(t, &GatlingRunResponse{Status: 3})
		state := RunState{RunId: "run-1", LastState: 3}
		result, err := action.Status(context.Background(), &state)
		require.NoError(t, err)
		assert.Nil(t, result.Messages)
	})

	t.Run("completes with an error when the run reports one", func(t *testing.T) {
		withMockApi(t, &GatlingRunResponse{Status: 2, Error: "boom"})
		state := RunState{RunId: "run-1", LastState: 2}
		result, err := action.Status(context.Background(), &state)
		require.NoError(t, err)
		assert.True(t, result.Completed)
		require.NotNil(t, result.Error)
		assert.Equal(t, action_kit_api.Errored, *result.Error.Status)
		assert.Equal(t, "Simulation error: boom", result.Error.Title)
	})

	t.Run("completes successfully", func(t *testing.T) {
		withMockApi(t, &GatlingRunResponse{Status: 4})
		state := RunState{RunId: "run-1", LastState: 3}
		result, err := action.Status(context.Background(), &state)
		require.NoError(t, err)
		assert.True(t, result.Completed)
		assert.Nil(t, result.Error)
	})

	t.Run("fails the action on failed assertions", func(t *testing.T) {
		withMockApi(t, &GatlingRunResponse{Status: 8})
		state := RunState{RunId: "run-1", LastState: 3}
		result, err := action.Status(context.Background(), &state)
		require.NoError(t, err)
		assert.True(t, result.Completed)
		require.NotNil(t, result.Error)
		assert.Equal(t, action_kit_api.Failed, *result.Error.Status)
		assert.Equal(t, "Simulation ended: AssertionsFailed", result.Error.Title)
	})

	t.Run("errors the action on a broken run", func(t *testing.T) {
		withMockApi(t, &GatlingRunResponse{Status: 10})
		state := RunState{RunId: "run-1", LastState: 3}
		result, err := action.Status(context.Background(), &state)
		require.NoError(t, err)
		assert.True(t, result.Completed)
		require.NotNil(t, result.Error)
		assert.Equal(t, action_kit_api.Errored, *result.Error.Status)
		assert.Equal(t, "Simulation ended: BuildFailed", result.Error.Title)
	})
}

func TestStop(t *testing.T) {
	action := RunAction{}

	t.Run("does nothing without a run id", func(t *testing.T) {
		result, err := action.Stop(context.Background(), &RunState{})
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("fails when the run cannot be fetched", func(t *testing.T) {
		withMockApi(t, nil)
		_, err := action.Stop(context.Background(), &RunState{RunId: "run-1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Failed to get run info")
	})

	t.Run("aborts a run that is still going", func(t *testing.T) {
		aborted := withMockApi(t, &GatlingRunResponse{Status: 3})
		result, err := action.Stop(context.Background(), &RunState{RunId: "run-1"})
		require.NoError(t, err)
		assert.True(t, *aborted)
		assert.Empty(t, *result.Messages)
	})

	t.Run("reports assertions of a finished run without aborting", func(t *testing.T) {
		aborted := withMockApi(t, &GatlingRunResponse{
			Status: 8,
			Assertions: []GatlingRunAssertion{
				{Message: "p95 < 500ms", Result: true, ActualValue: 250},
				{Message: "errors == 0", Result: false, ActualValue: 3},
			},
		})
		result, err := action.Stop(context.Background(), &RunState{RunId: "run-1"})
		require.NoError(t, err)
		assert.False(t, *aborted)
		messages := *result.Messages
		require.Len(t, messages, 3)
		assert.Equal(t, "### Assertions", messages[0].Message)
		assert.Equal(t, "- ✅ p95 < 500ms (250)", messages[1].Message)
		assert.Equal(t, "- ❌ errors == 0 (3)", messages[2].Message)
	})
}
