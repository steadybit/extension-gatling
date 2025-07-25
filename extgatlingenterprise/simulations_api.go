// Copyright 2025 steadybit GmbH. All rights reserved.

package extgatlingenterprise

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-gatling/config"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"io"
	"net/http"
	"net/url"
)

type GatlingStartSimulationRequest struct {
	Title                     string            `json:"title"`
	Description               string            `json:"description"`
	ExtraSystemProperties     map[string]string `json:"extraSystemProperties"`
	ExtraEnvironmentVariables map[string]string `json:"extraEnvironmentVariables"`
}
type GatlingRunAssertion struct {
	Message     string  `json:"message"`
	Result      bool    `json:"result"`
	ActualValue float64 `json:"actualValue"`
}
type GatlingRunResponse struct {
	Status      int                   `json:"status"`
	Error       string                `json:"error"`
	DeployStart int64                 `json:"deployStart"`
	DeployEnd   int64                 `json:"deployEnd"`
	InjectStart int64                 `json:"injectStart"`
	InjectEnd   int64                 `json:"injectEnd"`
	Assertions  []GatlingRunAssertion `json:"assertions"`
}

type GatlingStartResponse struct {
	ClassName   string `json:"className"`
	RunId       string `json:"runId"`
	ReportsPath string `json:"reportsPath"`
	RunsPath    string `json:"runsPath"`
}

type GatlingSimulationBuild struct {
	PkgId string `json:"pkgId"`
}

type GatlingSimulation struct {
	Id        string                 `json:"id"`
	Name      string                 `json:"name"`
	TeamId    string                 `json:"teamId"`
	ClassName string                 `json:"className"`
	Build     GatlingSimulationBuild `json:"build"`
}

func GetSimulations() []GatlingSimulation {
	var specification = config.Config
	var apiToken = specification.EnterpriseApiToken
	simulationsUrl, err := url.Parse(specification.EnterpriseApiBaseUrl)
	if err != nil {
		log.Error().Msgf("Failed to parse gatling base url. Got error: %s", err)
		return nil
	}

	client := getClient()
	simulationsUrl.Path += "/simulations"
	req, err := http.NewRequest("GET", simulationsUrl.String(), nil)
	if err != nil {
		return nil
	}
	req.Header.Add("Authorization", apiToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", fmt.Sprintf("steadybit-extension-gatling/%s", extbuild.GetSemverVersionStringOrUnknown()))

	response, err := client.Do(req)
	if err != nil {
		log.Error().Msgf("Failed to get simulations from gatling enterprise api. Got error: %s", err)
		return nil
	}
	if response.StatusCode != 200 {
		log.Error().Msgf("Failed to get simulations from gatling enterprise api. Got status code: %s", response.Status)
		return nil
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Msgf("Failed to close response body. Got error: %s", err)
			return
		}
	}(response.Body)

	var result []GatlingSimulation
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		log.Error().Msgf("Failed to decode response body. Got error: %s", err)
		return nil
	}

	return result
}

func RunSimulation(simulationId string, title string, description string, systemProperties map[string]string, environmentVariables map[string]string) (*string, error) {
	var specification = config.Config
	var apiToken = specification.EnterpriseApiToken
	runSimulationUrl, err := url.Parse(specification.EnterpriseApiBaseUrl)
	if err != nil {
		log.Error().Msgf("Failed to parse gatling base url. Got error: %s", err)
		return nil, err
	}

	client := getClient()
	runSimulationUrl.Path += "/simulations/start"
	q := runSimulationUrl.Query()
	q.Add("simulation", simulationId)
	runSimulationUrl.RawQuery = q.Encode()

	body := &GatlingStartSimulationRequest{
		Title:                     title,
		Description:               description,
		ExtraSystemProperties:     systemProperties,
		ExtraEnvironmentVariables: environmentVariables,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		log.Error().Msgf("Failed to prepare start request. Got error: %s", err)
		return nil, err
	}

	log.Debug().Str("url", runSimulationUrl.String()).Str("body", string(bodyBytes)).Msg("Starting gatling simulation....")
	req, err := http.NewRequest("POST", runSimulationUrl.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", apiToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", fmt.Sprintf("steadybit-extension-gatling/%s", extbuild.GetSemverVersionStringOrUnknown()))

	response, err := client.Do(req)
	if err != nil {
		log.Error().Msgf("Failed to start simulation via gatling enterprise api. Got error: %s", err)
		return nil, err
	}
	if response.StatusCode != 200 {
		log.Error().Msgf("Failed to start simulation via gatling enterprise api. Got status code: %s", response.Status)
		return nil, fmt.Errorf("failed to start simulation via gatling enterprise api. Got status code: %s", response.Status)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Msgf("Failed to close response body. Got error: %s", err)
			return
		}
	}(response.Body)

	var result GatlingStartResponse
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		log.Error().Msgf("Failed to decode response body. Got error: %s", err)
		return nil, err
	}

	log.Info().Msgf("Successfully started simulation: %+v", result)

	return extutil.Ptr(result.RunId), nil
}

func GetRun(runId string) (*GatlingRunResponse, error) {
	var specification = config.Config
	var apiToken = specification.EnterpriseApiToken
	runUrl, err := url.Parse(specification.EnterpriseApiBaseUrl)
	if err != nil {
		log.Error().Msgf("Failed to parse gatling base url. Got error: %s", err)
		return nil, err
	}

	client := getClient()
	runUrl.Path += "/run"
	q := runUrl.Query()
	q.Add("run", runId)
	runUrl.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", runUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", apiToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", fmt.Sprintf("steadybit-extension-gatling/%s", extbuild.GetSemverVersionStringOrUnknown()))

	response, err := client.Do(req)
	if err != nil {
		log.Error().Msgf("Failed to get run via gatling enterprise api. Got error: %s", err)
		return nil, err
	}
	if response.StatusCode != 200 {
		log.Error().Msgf("Failed to get run via gatling enterprise api. Got status code: %s", response.Status)
		return nil, fmt.Errorf("failed to get run via gatling enterprise api. Got status code: %s", response.Status)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Msgf("Failed to close response body. Got error: %s", err)
			return
		}
	}(response.Body)

	var result GatlingRunResponse
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		log.Error().Msgf("Failed to decode response body. Got error: %s", err)
		return nil, err
	}

	return extutil.Ptr(result), nil
}

func getClient() *http.Client {
	if config.Config.InsecureSkipVerify {
		log.Debug().Msg("InsecureSkipVerify is enabled. This is not recommended for production use.")
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.Config.InsecureSkipVerify,
			},
		}
		return &http.Client{Transport: transport}
	}
	return &http.Client{}
}

func StopRun(runId string) error {
	var specification = config.Config
	var apiToken = specification.EnterpriseApiToken
	abortRunUrl, err := url.Parse(specification.EnterpriseApiBaseUrl)
	if err != nil {
		log.Error().Msgf("Failed to parse gatling base url. Got error: %s", err)
		return err
	}

	client := getClient()
	abortRunUrl.Path += "/simulations/abort"
	q := abortRunUrl.Query()
	q.Add("run", runId)
	abortRunUrl.RawQuery = q.Encode()

	req, err := http.NewRequest("POST", abortRunUrl.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", apiToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", fmt.Sprintf("steadybit-extension-gatling/%s", extbuild.GetSemverVersionStringOrUnknown()))

	response, err := client.Do(req)
	if err != nil {
		log.Error().Msgf("Failed to abort run via gatling enterprise api. Got error: %s", err)
		return err
	}
	if response.StatusCode != 200 {
		log.Error().Msgf("Failed to abort run via gatling enterprise api. Got status code: %s", response.Status)
		return fmt.Errorf("failed to abort run via gatling enterprise api. Got status code: %s", response.Status)
	}
	return nil
}
