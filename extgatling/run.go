/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extgatling

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extcmd"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extfile"
	"github.com/steadybit/extension-kit/extutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type GatlingLoadTestRunAction struct{}

type GatlingLoadTestRunState struct {
	Command     []string  `json:"command"`
	Pid         int       `json:"pid"`
	CmdStateID  string    `json:"cmdStateId"`
	ExecutionId uuid.UUID `json:"executionId"`
}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[GatlingLoadTestRunState]           = (*GatlingLoadTestRunAction)(nil)
	_ action_kit_sdk.ActionWithStatus[GatlingLoadTestRunState] = (*GatlingLoadTestRunAction)(nil)
	_ action_kit_sdk.ActionWithStop[GatlingLoadTestRunState]   = (*GatlingLoadTestRunAction)(nil)
)

func NewGatlingLoadTestRunAction() action_kit_sdk.Action[GatlingLoadTestRunState] {
	return &GatlingLoadTestRunAction{}
}

func (l *GatlingLoadTestRunAction) NewEmptyState() GatlingLoadTestRunState {
	return GatlingLoadTestRunState{}
}

func (l *GatlingLoadTestRunAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          actionId,
		Label:       "Gatling",
		Description: "Execute a Gatling load test.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(actionIcon),
		Kind:        action_kit_api.LoadTest,
		TimeControl: action_kit_api.TimeControlInternal,
		Hint: &action_kit_api.ActionHint{
			Content: "Please note that load tests are executed by the gatling extension participating in the experiment, consuming resources of the system that it is installed in.",
			Type:    action_kit_api.HintWarning,
		},
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:        "file",
				Label:       "Gatling Sources",
				Description: extutil.Ptr("Upload your Gatling Sources. zip files will be extracted."),
				Type:        action_kit_api.File,
				Required:    extutil.Ptr(true),
				AcceptedFileTypes: extutil.Ptr([]string{
					".zip",
					".java",
					".scala",
					".kt",
				}),
			},
			{
				Name:        "parameter",
				Label:       "Parameter",
				Description: extutil.Ptr("Parameters will be accessible from your Gatling Source via Java System Properties, e.g. System.getProperty(\"myParameter\")"),
				Type:        action_kit_api.KeyValue,
				Required:    extutil.Ptr(true),
			},
			{
				Name:        "simulation",
				Label:       "Simulation",
				Description: extutil.Ptr("ClassName of the Simulation to execute. Can be omitted if there is only one simulation in the source files."),
				Type:        action_kit_api.String,
				Required:    extutil.Ptr(false),
			},
		},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("5s"),
		}),
		Stop: extutil.Ptr(action_kit_api.MutatingEndpointReference{}),
	}
}

type GatlingLoadTestRunConfig struct {
	Parameter  []map[string]string
	File       string
	Simulation string
}

func (l *GatlingLoadTestRunAction) Prepare(_ context.Context, state *GatlingLoadTestRunState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	var config GatlingLoadTestRunConfig
	if err := extconversion.Convert(request.Config, &config); err != nil {
		return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
	}
	executionRoot := fmt.Sprintf("/tmp/steadybit/%v", request.ExecutionId) //Folder is managed by action_kit_sdk's file download handling
	reportFolder := fmt.Sprintf("%v/report", executionRoot)
	if err := os.Mkdir(reportFolder, 0755); err != nil {
		return nil, extension_kit.ToError("Failed to create report folder.", err)
	}
	srcFolder := fmt.Sprintf("%v/src", executionRoot)
	if err := os.Mkdir(srcFolder, 0755); err != nil {
		return nil, extension_kit.ToError("Failed to create src folder.", err)
	}

	if filepath.Ext(config.File) == ".zip" {
		log.Info().Msgf("Unzip with command: %s %s %s %s", "unzip", config.File, "-d", srcFolder)
		if err := exec.Command("unzip", config.File, "-d", srcFolder).Run(); err != nil {
			return nil, extension_kit.ToError("Failed to unzip file.", err)
		}
	} else {
		if err := exec.Command("mv", config.File, srcFolder).Run(); err != nil {
			return nil, extension_kit.ToError("Failed to move file.", err)
		}
	}

	command := []string{
		"gatling.sh",
		"-rm",
		"local",
		"-rd",
		"\"gatling execution by steadybit\"",
		"-sf",
		srcFolder,
		"-rf",
		reportFolder,
		"-bf",
		srcFolder,
		"-rsf",
		srcFolder,
	}
	if config.Simulation != "" {
		command = append(command, "-s")
		command = append(command, config.Simulation)
	}
	if config.Parameter != nil {
		command = append(command, "-erjo")
		var properties string
		for _, value := range config.Parameter {
			properties += fmt.Sprintf("-D%v=%v ", value["key"], value["value"])
		}
		command = append(command, properties)
	}

	state.ExecutionId = request.ExecutionId
	state.Command = command

	return nil, nil
}

func (l *GatlingLoadTestRunAction) Start(_ context.Context, state *GatlingLoadTestRunState) (*action_kit_api.StartResult, error) {
	log.Info().Msgf("Starting Gatling load test with command: %s", strings.Join(state.Command, " "))
	cmd := exec.Command(state.Command[0], state.Command[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmdState := extcmd.NewCmdState(cmd)
	state.CmdStateID = cmdState.Id
	err := cmd.Start()
	if err != nil {
		return nil, extension_kit.ToError("Failed to start command.", err)
	}

	state.Pid = cmd.Process.Pid
	go func() {
		cmdErr := cmd.Wait()
		if cmdErr != nil {
			log.Error().Msgf("Failed to execute gatling: %s", cmdErr)
		}
	}()
	log.Info().Msgf("Started load test.")

	state.Command = nil
	return nil, nil
}

func (l *GatlingLoadTestRunAction) Status(_ context.Context, state *GatlingLoadTestRunState) (*action_kit_api.StatusResult, error) {
	log.Debug().Msgf("Checking Gatling status for %d\n", state.Pid)

	cmdState, err := extcmd.GetCmdState(state.CmdStateID)
	if err != nil {
		return nil, extension_kit.ToError("Failed to find command state", err)
	}

	var result action_kit_api.StatusResult

	// check if gatling is still running
	exitCode := cmdState.Cmd.ProcessState.ExitCode()
	stdOut := cmdState.GetLines(false)
	stdOutToLog(stdOut)
	if exitCode == -1 {
		log.Debug().Msgf("Gatling is still running")
		result.Completed = false
	} else if exitCode == 0 {
		log.Info().Msgf("Gatling run completed successfully")
		result.Completed = true
	} else {
		title := fmt.Sprintf("Gatling run failed, exit-code %d", exitCode)
		result.Completed = true
		result.Error = &action_kit_api.ActionKitError{
			Status: extutil.Ptr(action_kit_api.Errored),
			Title:  title,
		}
	}

	messages := stdOutToMessages(stdOut)
	log.Debug().Msgf("Returning %d messages", len(messages))

	result.Messages = extutil.Ptr(messages)
	return &result, nil
}

func (l *GatlingLoadTestRunAction) Stop(_ context.Context, state *GatlingLoadTestRunState) (*action_kit_api.StopResult, error) {
	if state.CmdStateID == "" {
		log.Info().Msg("Gatling not yet started, nothing to stop.")
		return nil, nil
	}

	cmdState, err := extcmd.GetCmdState(state.CmdStateID)
	if err != nil {
		return nil, extension_kit.ToError("Failed to find command state", err)
	}
	extcmd.RemoveCmdState(state.CmdStateID)

	// kill Gatling if it is still running
	gracefulKill(state.Pid, cmdState)

	// read Stout and Stderr and send it as Messages
	stdOut := cmdState.GetLines(true)
	stdOutToLog(stdOut)
	messages := stdOutToMessages(stdOut)

	// read return code and send it as Message
	exitCode := cmdState.Cmd.ProcessState.ExitCode()
	if exitCode != 0 && exitCode != -1 {
		messages = append(messages, action_kit_api.Message{
			Level:   extutil.Ptr(action_kit_api.Error),
			Message: fmt.Sprintf("Gatling run failed with exit code %d", exitCode),
		})
	}

	var artifacts []action_kit_api.Artifact
	executionRoot := fmt.Sprintf("/tmp/steadybit/%v", state.ExecutionId) //Folder is managed by action_kit_sdk's file download handling
	reportFolder := fmt.Sprintf("%v/report", executionRoot)
	files, err := os.ReadDir(reportFolder)
	if err != nil {
		return nil, extension_kit.ToError("Failed to read report folder", err)
	}

	for _, file := range files {
		if file.IsDir() {
			simulationLog := fmt.Sprintf("%v/%v/simulation.log", reportFolder, file.Name())
			_, err = os.Stat(simulationLog)
			if err == nil { // file exists
				zippedReport := fmt.Sprintf("%v/report.zip", reportFolder)
				log.Info().Msgf("Zip report with command: %s %s %s %s", "zip", "-r", zippedReport, ".")
				zipCommand := exec.Command("zip", "-r", zippedReport, ".")
				zipCommand.Dir = fmt.Sprintf("%v/%v", reportFolder, file.Name())
				zipErr := zipCommand.Run()
				if zipErr != nil {
					return nil, extension_kit.ToError("Failed to zip report", err)
				}
				content, err := extfile.File2Base64(zippedReport)
				if err != nil {
					return nil, err
				}
				artifacts = append(artifacts, action_kit_api.Artifact{
					Label: "$(experimentKey)_$(executionId)_report.zip",
					Data:  content,
				})
			}
		}
	}

	log.Debug().Msgf("Returning %d messages", len(messages))
	return &action_kit_api.StopResult{
		Artifacts: extutil.Ptr(artifacts),
		Messages:  extutil.Ptr(messages),
	}, nil
}

func gracefulKill(pid int, cmdState *extcmd.CmdState) {
	exitCode := cmdState.Cmd.ProcessState.ExitCode()
	if exitCode == -1 {

		log.Info().Msg("Gatling process running - send SIGINT.")
		_ = syscall.Kill(-pid, syscall.SIGINT)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Gatling process still running - send SIGKILL.")
				_ = syscall.Kill(-pid, syscall.SIGKILL)
				return
			case <-time.After(1000 * time.Millisecond):
				exitCode = cmdState.Cmd.ProcessState.ExitCode()
				if exitCode != -1 {
					log.Info().Msg("Gatling process stopped (SIGINT).")
					return
				}
			}
		}
	}
}
