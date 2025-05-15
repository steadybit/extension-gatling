// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extgatlingenterprise

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-gatling/config"
	"github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"strconv"
)

type RunAction struct {
}

type RunState struct {
	ExperimentKey string `json:"experimentKey"`
	ExecutionId   int    `json:"executionId"`
	SimulationId  string `json:"simulationId"`
	RunId         string `json:"runId"`
	LastState     int    `json:"lastState"`
}

func NewGatlingEnterpriseRunAction() action_kit_sdk.Action[RunState] {
	return RunAction{}
}

// Make sure PostmanAction implements all required interfaces
var _ action_kit_sdk.Action[RunState] = (*RunAction)(nil)
var _ action_kit_sdk.ActionWithStatus[RunState] = (*RunAction)(nil)
var _ action_kit_sdk.ActionWithStop[RunState] = (*RunAction)(nil)

func (f RunAction) NewEmptyState() RunState {
	return RunState{}
}

func (f RunAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          actionId,
		Label:       "Gatling Enterprise",
		Description: "Run a simulation via Gatling Enterprise",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Kind:        action_kit_api.LoadTest,
		Icon:        extutil.Ptr(actionIcon),
		Technology:  extutil.Ptr("Gatling"),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          targetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.ExactlyOne),
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label: "simulation name",
					Query: "gatling.simulation.name=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlInternal,
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Estimated duration",
				DefaultValue: extutil.Ptr("30s"),
				Description:  extutil.Ptr("The step will run as long as needed. You can set this estimation to size the step in the experiment editor for a better understanding of the time schedule."),
				Required:     extutil.Ptr(true),
				Type:         action_kit_api.Duration,
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status:  extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{}),
		Stop:    extutil.Ptr(action_kit_api.MutatingEndpointReference{}),
		Widgets: extutil.Ptr([]action_kit_api.Widget{
			action_kit_api.MarkdownWidget{
				Type:        action_kit_api.ComSteadybitWidgetMarkdown,
				Title:       "Gatling Enterprise",
				MessageType: "GATLING",
				Append:      true,
			},
		}),
	}
}

func (f RunAction) Prepare(_ context.Context, state *RunState, raw action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	simulationIds := raw.Target.Attributes["gatling.simulation.id"]
	if len(simulationIds) == 0 {
		return nil, extension_kit.ToError("No simulation id provided", nil)
	}
	if len(simulationIds) > 1 {
		return nil, extension_kit.ToError("More than one simulation id provided", nil)
	}
	state.SimulationId = simulationIds[0]
	if raw.ExecutionContext.ExecutionId != nil {
		state.ExecutionId = *raw.ExecutionContext.ExecutionId
	}
	if raw.ExecutionContext.ExperimentKey != nil {
		state.ExperimentKey = *raw.ExecutionContext.ExperimentKey
	}
	state.LastState = -1
	return nil, nil
}

func (f RunAction) Start(_ context.Context, state *RunState) (*action_kit_api.StartResult, error) {
	runId, err := RunSimulation(state.SimulationId, fmt.Sprintf("Steadybit - %s - %d", state.ExperimentKey, state.ExecutionId), fmt.Sprintf("Executed by Steadybit Experiment %s, Execution %d", state.ExperimentKey, state.ExecutionId))
	if err != nil {
		return nil, extension_kit.ToError("Failed to run simulation", err)
	}
	state.RunId = *runId

	result := &action_kit_api.StartResult{
		Messages: &[]action_kit_api.Message{
			{
				Message: fmt.Sprintf("[Open Summary](https://cloud.gatling.io/o/%s/simulations/%s/runs/%s)", config.Config.EnterpriseOrganizationSlug, state.SimulationId, state.RunId),
				Type:    extutil.Ptr("GATLING"),
			},
			{
				Message: fmt.Sprintf("[Open Report](https://cloud.gatling.io/o/%s/simulations/%s/runs/%s/details)", config.Config.EnterpriseOrganizationSlug, state.SimulationId, state.RunId),
				Type:    extutil.Ptr("GATLING"),
			},
			{
				Message: fmt.Sprintf("[Open Logs](https://cloud.gatling.io/o/%s/simulations/%s/runs/%s/logs)", config.Config.EnterpriseOrganizationSlug, state.SimulationId, state.RunId),
				Type:    extutil.Ptr("GATLING"),
			},
		},
	}
	return result, nil
}

func (f RunAction) Status(_ context.Context, state *RunState) (*action_kit_api.StatusResult, error) {
	log.Info().Str("runId", state.RunId).Msg("Checking run status.")

	run, err := GetRun(state.RunId)
	if err != nil {
		return nil, extension_kit.ToError("Failed to get run info", err)
	}

	var result action_kit_api.StatusResult

	if state.LastState != run.Status {
		log.Info().Str("state", statusToString(run.Status)).Msg("Simulation state changed")
		result.Messages = extutil.Ptr([]action_kit_api.Message{
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Message: fmt.Sprintf("Simulation state: %s", statusToString(run.Status)),
			},
			{
				Message: "- " + statusToString(run.Status) + appendDots(run.Status),
				Type:    extutil.Ptr("GATLING"),
			},
		})
		state.LastState = run.Status
	}

	if run.Error != "" {
		log.Info().Str("error", run.Error).Msg("Simulation error")
		result.Error = &action_kit_api.ActionKitError{
			Status: extutil.Ptr(action_kit_api.Errored),
			Title:  fmt.Sprintf("Simulation error: %s", run.Error),
		}
		result.Completed = true
		return &result, nil
	}

	if run.Status >= 4 {
		log.Info().Str("state", statusToString(run.Status)).Msg("Simulation ended")
		result.Completed = true
		if run.Status >= 9 && run.Status <= 13 {
			result.Error = &action_kit_api.ActionKitError{
				Status: extutil.Ptr(action_kit_api.Errored),
				Title:  fmt.Sprintf("Simulation ended: %s", statusToString(run.Status)),
			}
		}
		if run.Status == 8 {
			result.Error = &action_kit_api.ActionKitError{
				Status: extutil.Ptr(action_kit_api.Failed),
				Title:  fmt.Sprintf("Simulation ended: %s", statusToString(run.Status)),
			}
		}
	}

	return &result, nil
}

func appendDots(status int) string {
	if status <= 3 {
		return "..."
	}
	return ""
}

func statusToString(status int) string {
	switch status {
	case 0:
		return "Building"
	case 1:
		return "Deploying"
	case 2:
		return "Deployed"
	case 3:
		return "Injecting"
	case 4:
		return "Successful"
	case 5:
		return "AssertionsSuccessful"
	case 6:
		return "AutomaticallyStopped"
	case 7:
		return "Stopped"
	case 8:
		return "AssertionsFailed"
	case 9:
		return "Timeout"
	case 10:
		return "BuildFailed"
	case 11:
		return "Broken"
	case 12:
		return "DeploymentFailed"
	case 13:
		return "InsufficientLicense"
	case 15:
		return "StopRequested"
	default:
		return "Unknown (" + strconv.Itoa(status) + ")"
	}
}

func (f RunAction) Stop(_ context.Context, state *RunState) (*action_kit_api.StopResult, error) {
	if state.RunId == "" {
		return nil, nil
	}

	run, err := GetRun(state.RunId)
	if err != nil {
		return nil, extension_kit.ToError("Failed to get run info", err)
	}

	messages := make([]action_kit_api.Message, 0)
	if len(run.Assertions) > 0 {
		messages = append(messages, action_kit_api.Message{
			Message: "### Assertions",
			Type:    extutil.Ptr("GATLING"),
		})
		for _, assertion := range run.Assertions {
			icon := "❌"
			if assertion.Result {
				icon = "✅"
			}
			messages = append(messages, action_kit_api.Message{
				Message: fmt.Sprintf("- %s %s (%.0f)", icon, assertion.Message, assertion.ActualValue),
				Type:    extutil.Ptr("GATLING"),
			})
		}
	}

	if run.Status < 4 {
		log.Info().Str("runId", state.RunId).Msgf("Stop run")
		abortErr := StopRun(state.RunId)
		if abortErr != nil {
			return nil, extension_kit.ToError("Failed to abort run", abortErr)
		}
	} else {
		log.Debug().Str("runId", state.RunId).Msgf("Already stopped")
	}

	return &action_kit_api.StopResult{Messages: extutil.Ptr(messages)}, nil
}
