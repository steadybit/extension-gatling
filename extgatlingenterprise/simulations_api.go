package extgatlingenterprise

import (
	"bytes"
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

type GatlingRunAssertion struct {
	Message     string `json:"message"`
	Result      bool   `json:"result"`
	ActualValue string `json:"actualValue"`
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

	client := &http.Client{}
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

func RunSimulation(simulationId string, title string, description string) (*string, error) {
	var specification = config.Config
	var apiToken = specification.EnterpriseApiToken
	runSimulationUrl, err := url.Parse(specification.EnterpriseApiBaseUrl)
	if err != nil {
		log.Error().Msgf("Failed to parse gatling base url. Got error: %s", err)
		return nil, err
	}

	client := &http.Client{}
	runSimulationUrl.Path += "/simulations/start"
	q := runSimulationUrl.Query()
	q.Add("simulation", simulationId)
	runSimulationUrl.RawQuery = q.Encode()

	body := fmt.Sprintf("{\"title\":\"%s\",\"description\":\"%s\"}", title, description)
	log.Debug().Str("url", runSimulationUrl.String()).Str("body", body).Msg("Starting gatling simulation....")
	req, err := http.NewRequest("POST", runSimulationUrl.String(), bytes.NewBufferString(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", apiToken)
	req.Header.Add("Accept", "application/json")
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

	client := &http.Client{}
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

func StopRun(runId string) error {
	var specification = config.Config
	var apiToken = specification.EnterpriseApiToken
	abortRunUrl, err := url.Parse(specification.EnterpriseApiBaseUrl)
	if err != nil {
		log.Error().Msgf("Failed to parse gatling base url. Got error: %s", err)
		return err
	}

	client := &http.Client{}
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
