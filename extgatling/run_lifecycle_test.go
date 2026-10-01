/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package extgatling

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-gatling/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newExecutionRoot creates the folder the action_kit_sdk would create for downloaded files of an execution.
func newExecutionRoot(t *testing.T) (uuid.UUID, string) {
	t.Helper()
	executionId := uuid.New()
	root := fmt.Sprintf("/tmp/steadybit/%v", executionId)
	require.NoError(t, os.MkdirAll(root, 0755))
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return executionId, root
}

// prepareInRepoRoot runs Prepare from the repository root, where the gatling-maven-scaffold folder lives.
func prepareInRepoRoot(t *testing.T, executionId uuid.UUID, cfg map[string]any) (*GatlingLoadTestRunState, *action_kit_api.PrepareResult, error) {
	t.Helper()
	t.Chdir("..")
	state := &GatlingLoadTestRunState{}
	result, err := (&GatlingLoadTestRunAction{}).Prepare(context.Background(), state, action_kit_api.PrepareActionRequestBody{
		ExecutionId: executionId,
		Config:      cfg,
		ExecutionContext: &action_kit_api.ExecutionContext{
			ExperimentKey: new("ADM-1"),
			ExecutionId:   new(7),
		},
	})
	return state, result, err
}

func TestPrepare_JavaSource(t *testing.T) {
	executionId, root := newExecutionRoot(t)
	source := filepath.Join(root, "BasicSimulation.java")
	writeFile(t, source, "class BasicSimulation {}")

	state, result, err := prepareInRepoRoot(t, executionId, map[string]any{
		"file":       source,
		"simulation": "BasicSimulation",
		"parameter":  []map[string]string{{"key": "users", "value": "10"}},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Equal(t, executionId, state.ExecutionId)
	assert.Equal(t, []string{
		"mvn",
		"integration-test",
		"-o",
		"-Dgatling.runDescription=\"executed by Steadybit - Experiment ADM-1 - Execution 7\"",
		"-Dgatling.simulationClass=BasicSimulation",
		"-Dusers=10 ",
	}, state.Command)
	assert.FileExists(t, filepath.Join(root, "gatling-maven-scaffold", "src", "test", "java", "BasicSimulation.java"))
}

func TestPrepare_ZippedSources(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		folder  string
		profile string
	}{
		{name: "kotlin", file: "simulations/BasicSimulation.kt", folder: "kotlin", profile: "-Pkotlin"},
		{name: "scala", file: "simulations/BasicSimulation.scala", folder: "scala", profile: "-Pscala"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			executionId, root := newExecutionRoot(t)
			archive := writeArchive(t, map[string]string{tc.file: "simulation"})

			state, result, err := prepareInRepoRoot(t, executionId, map[string]any{"file": archive})

			require.NoError(t, err)
			assert.Nil(t, result)
			assert.Equal(t, tc.profile, state.Command[len(state.Command)-1])
			assert.Len(t, state.Command, 5, "no simulation class and no parameters were configured")
			assert.FileExists(t, filepath.Join(root, "gatling-maven-scaffold", "src", "test", tc.folder, filepath.FromSlash(tc.file)))
		})
	}
}

func TestPrepare_FailsWithoutSourceFiles(t *testing.T) {
	executionId, _ := newExecutionRoot(t)
	archive := writeArchive(t, map[string]string{"README.md": "no simulation in here"})

	_, _, err := prepareInRepoRoot(t, executionId, map[string]any{"file": archive})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "No source files found.")
}

func TestPrepare_FailsWhenTheSourceCannotBeMoved(t *testing.T) {
	executionId, _ := newExecutionRoot(t)

	_, _, err := prepareInRepoRoot(t, executionId, map[string]any{"file": "/does/not/exist.java"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to move file.")
}

func TestPrepare_FailsWithInvalidConfig(t *testing.T) {
	_, err := (&GatlingLoadTestRunAction{}).Prepare(context.Background(), &GatlingLoadTestRunState{}, action_kit_api.PrepareActionRequestBody{
		Config: map[string]any{"file": []string{"not", "a", "string"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to unmarshal the config.")
}

// startScript starts the given shell script in place of maven inside a prepared execution root.
func startScript(t *testing.T, script string) (*GatlingLoadTestRunState, string) {
	t.Helper()
	executionId, root := newExecutionRoot(t)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "gatling-maven-scaffold"), 0755))
	state := &GatlingLoadTestRunState{
		ExecutionId: executionId,
		Command:     []string{"sh", "-c", script},
	}

	result, err := (&GatlingLoadTestRunAction{}).Start(context.Background(), state)

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Nil(t, state.Command, "the command is not needed anymore once started")
	assert.NotZero(t, state.Pid)
	assert.NotEmpty(t, state.CmdStateID)
	return state, root
}

func awaitCompletion(t *testing.T, state *GatlingLoadTestRunState) *action_kit_api.StatusResult {
	t.Helper()
	var result *action_kit_api.StatusResult
	require.Eventually(t, func() bool {
		var err error
		result, err = (&GatlingLoadTestRunAction{}).Status(context.Background(), state)
		require.NoError(t, err)
		return result.Completed
	}, 10*time.Second, 50*time.Millisecond)
	return result
}

func TestStatus_ReportsTheExitCode(t *testing.T) {
	cases := []struct {
		name   string
		exit   int
		status *action_kit_api.ActionKitErrorStatus
		title  string
	}{
		{name: "success", exit: 0},
		{name: "failing assertions", exit: 2, status: new(action_kit_api.Failed), title: "Gatling run ended with failing assertions. Reports are attached."},
		{name: "error", exit: 1, status: new(action_kit_api.Errored), title: "Gatling run errored, exit-code 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state, _ := startScript(t, fmt.Sprintf("echo simulation finished; exit %d", tc.exit))

			result := awaitCompletion(t, state)

			if tc.status == nil {
				assert.Nil(t, result.Error)
			} else {
				require.NotNil(t, result.Error)
				assert.Equal(t, tc.status, result.Error.Status)
				assert.Equal(t, tc.title, result.Error.Title)
			}
		})
	}
}

// interruptibleScript keeps running until it receives SIGINT and then exits like gatling does when cancelled.
// It creates a marker once the trap is installed, as a SIGINT arriving earlier would kill the shell instead.
// The background sleep ignores SIGINT and must not hold on to stdout, otherwise the command only finishes
// once the sleep does.
const interruptibleScript = "trap 'kill $!; exit 130' INT; touch ready; sleep 30 >/dev/null 2>&1 & wait"

func startInterruptible(t *testing.T) *GatlingLoadTestRunState {
	t.Helper()
	state, root := startScript(t, interruptibleScript)
	require.Eventually(t, func() bool {
		_, err := os.Stat(filepath.Join(root, "gatling-maven-scaffold", "ready"))
		return err == nil
	}, 10*time.Second, 10*time.Millisecond)
	return state
}

func TestStatus_StillRunning(t *testing.T) {
	state := startInterruptible(t)
	t.Cleanup(func() { _, _ = (&GatlingLoadTestRunAction{}).Stop(context.Background(), state) })

	result, err := (&GatlingLoadTestRunAction{}).Status(context.Background(), state)

	require.NoError(t, err)
	assert.False(t, result.Completed)
	assert.Nil(t, result.Error)
}

func TestStatus_FailsForUnknownCommand(t *testing.T) {
	_, err := (&GatlingLoadTestRunAction{}).Status(context.Background(), &GatlingLoadTestRunState{CmdStateID: "unknown"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to find command state")
}

func TestStop_FailsForUnknownCommand(t *testing.T) {
	_, err := (&GatlingLoadTestRunAction{}).Stop(context.Background(), &GatlingLoadTestRunState{CmdStateID: "unknown"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to find command state")
}

func TestStop_AttachesTheReports(t *testing.T) {
	state, root := startScript(t, "echo done; exit 2")
	awaitCompletion(t, state)
	report := filepath.Join(root, "gatling-maven-scaffold", "target", "gatling", "basicsimulation-20260824180012345")
	writeFile(t, filepath.Join(report, "simulation.log"), "RUN\n")
	writeFile(t, filepath.Join(report, "index.html"), "<html></html>")

	result, err := (&GatlingLoadTestRunAction{}).Stop(context.Background(), state)

	require.NoError(t, err)
	require.NotNil(t, result.Error)
	assert.Equal(t, new(action_kit_api.Failed), result.Error.Status)
	require.Len(t, *result.Artifacts, 1)
	assert.Equal(t, "$(experimentKey)_$(executionId)_basicsimulation-20260824180012345_report.zip", (*result.Artifacts)[0].Label)
	assert.NotEmpty(t, (*result.Artifacts)[0].Data)
	assert.Contains(t, messageTexts(*result.Messages), "Gatling run stopped with exit code 2")
}

func TestStop_ErroredRunWithoutReport(t *testing.T) {
	state, root := startScript(t, "exit 3")
	awaitCompletion(t, state)

	result, err := (&GatlingLoadTestRunAction{}).Stop(context.Background(), state)

	require.NoError(t, err)
	require.NotNil(t, result.Error)
	assert.Equal(t, new(action_kit_api.Errored), result.Error.Status)
	assert.Equal(t, "Gatling run errored, exit-code 3", result.Error.Title)
	assert.Empty(t, *result.Artifacts)
	assert.Contains(t, messageTexts(*result.Messages), fmt.Sprintf("Found no gatling report below %s, so none is attached to this run.", root))
}

func TestStop_InterruptsARunningLoadTest(t *testing.T) {
	// 130 is what gatling exits with on SIGINT - a cancelled run is not an error.
	state := startInterruptible(t)

	result, err := (&GatlingLoadTestRunAction{}).Stop(context.Background(), state)

	require.NoError(t, err)
	assert.Nil(t, result.Error)
	assert.Contains(t, messageTexts(*result.Messages), "Gatling run stopped with exit code 130")
}

func TestDescribe_LocationSelection(t *testing.T) {
	originalConfig := config.Config
	t.Cleanup(func() { config.Config = originalConfig })

	config.Config.EnableLocationSelection = true
	description := (&GatlingLoadTestRunAction{}).Describe()
	require.NotNil(t, description.TargetSelection)
	assert.Equal(t, targetType, description.TargetSelection.TargetType)
	assert.Len(t, description.Parameters, 4)

	config.Config.EnableLocationSelection = false
	description = (&GatlingLoadTestRunAction{}).Describe()
	assert.Nil(t, description.TargetSelection)
	assert.Len(t, description.Parameters, 3)
}

func TestStdOutToLog(t *testing.T) {
	assert.NotPanics(t, func() { stdOutToLog([]string{"line 1\n", "  ", ""}) })
}

func messageTexts(messages []action_kit_api.Message) []string {
	texts := make([]string, 0, len(messages))
	for _, m := range messages {
		texts = append(texts, m.Message)
	}
	return texts
}
