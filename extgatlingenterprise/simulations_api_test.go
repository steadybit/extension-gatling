/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package extgatlingenterprise

import (
	"encoding/json"
	"fmt"
	"github.com/steadybit/extension-gatling/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSimulations(t *testing.T) {
	// Setup a mock server to handle the request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		assert.Equal(t, "/simulations", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Contains(t, r.Header.Get("User-Agent"), "steadybit-extension-gatling")

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		simulations := []GatlingSimulation{
			{
				Id:        "sim-123",
				Name:      "Test Simulation 1",
				TeamId:    "team-1",
				ClassName: "TestSimulation1",
				Build: GatlingSimulationBuild{
					PkgId: "pkg-1",
				},
			},
			{
				Id:        "sim-456",
				Name:      "Test Simulation 2",
				TeamId:    "team-1",
				ClassName: "TestSimulation2",
				Build: GatlingSimulationBuild{
					PkgId: "pkg-2",
				},
			},
		}
		json.NewEncoder(w).Encode(simulations)
	}))
	defer server.Close()

	// Save the original config and restore it after the test
	originalConfig := config.Config
	defer func() {
		config.Config = originalConfig
	}()

	// Set the config to use our test server
	config.Config.EnterpriseApiBaseUrl = server.URL
	config.Config.EnterpriseApiToken = "test-token"

	// Call the function under test
	simulations := GetSimulations()

	// Verify the results
	require.NotNil(t, simulations)
	require.Len(t, simulations, 2)
	assert.Equal(t, "sim-123", simulations[0].Id)
	assert.Equal(t, "Test Simulation 1", simulations[0].Name)
	assert.Equal(t, "TestSimulation1", simulations[0].ClassName)
	assert.Equal(t, "pkg-1", simulations[0].Build.PkgId)
	assert.Equal(t, "sim-456", simulations[1].Id)
	assert.Equal(t, "Test Simulation 2", simulations[1].Name)
}

func TestGetSimulations_Error(t *testing.T) {
	// Test cases for different error scenarios
	testCases := []struct {
		name             string
		serverStatus     int
		serverResponse   string
		expectedNilValue bool
	}{
		{
			name:             "API returns error status code",
			serverStatus:     http.StatusInternalServerError,
			serverResponse:   `{"error": "Internal server error"}`,
			expectedNilValue: true,
		},
		{
			name:             "API returns invalid JSON",
			serverStatus:     http.StatusOK,
			serverResponse:   `invalid json`,
			expectedNilValue: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup a mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.serverStatus)
				fmt.Fprint(w, tc.serverResponse)
			}))
			defer server.Close()

			// Save the original config and restore it after the test
			originalConfig := config.Config
			defer func() {
				config.Config = originalConfig
			}()

			// Set the config to use our test server
			config.Config.EnterpriseApiBaseUrl = server.URL
			config.Config.EnterpriseApiToken = "test-token"

			// Call the function under test
			simulations := GetSimulations()

			// Verify the results
			if tc.expectedNilValue {
				assert.Nil(t, simulations)
			} else {
				assert.NotNil(t, simulations)
			}
		})
	}
}

func TestRunSimulation(t *testing.T) {
	// Setup a mock server to handle the request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		assert.Equal(t, "/simulations/start", r.URL.Path)
		assert.Equal(t, "simulation=sim-123", r.URL.RawQuery)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		// Decode the request body to verify its contents
		var requestBody GatlingStartSimulationRequest
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&requestBody)
		require.NoError(t, err)

		assert.Equal(t, "Test Title", requestBody.Title)
		assert.Equal(t, "Test Description", requestBody.Description)
		assert.Equal(t, "value1", requestBody.ExtraSystemProperties["prop1"])
		assert.Equal(t, "value2", requestBody.ExtraSystemProperties["prop2"])
		assert.Equal(t, "envVal1", requestBody.ExtraEnvironmentVariables["env1"])

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := GatlingStartResponse{
			ClassName:   "TestSimulation",
			RunId:       "run-123",
			ReportsPath: "/reports/run-123",
			RunsPath:    "/runs/run-123",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Save the original config and restore it after the test
	originalConfig := config.Config
	defer func() {
		config.Config = originalConfig
	}()

	// Set the config to use our test server
	config.Config.EnterpriseApiBaseUrl = server.URL
	config.Config.EnterpriseApiToken = "test-token"

	// Call the function under test
	systemProperties := map[string]string{
		"prop1": "value1",
		"prop2": "value2",
	}
	environmentVariables := map[string]string{
		"env1": "envVal1",
	}
	runId, err := RunSimulation("sim-123", "Test Title", "Test Description", systemProperties, environmentVariables)

	// Verify the results
	require.NoError(t, err)
	require.NotNil(t, runId)
	assert.Equal(t, "run-123", *runId)
}

func TestRunSimulation_Error(t *testing.T) {
	// Test cases for different error scenarios
	testCases := []struct {
		name           string
		serverStatus   int
		serverResponse string
		expectError    bool
	}{
		{
			name:           "API returns error status code",
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `{"error": "Internal server error"}`,
			expectError:    true,
		},
		{
			name:           "API returns invalid JSON",
			serverStatus:   http.StatusOK,
			serverResponse: `invalid json`,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup a mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.serverStatus)
				fmt.Fprint(w, tc.serverResponse)
			}))
			defer server.Close()

			// Save the original config and restore it after the test
			originalConfig := config.Config
			defer func() {
				config.Config = originalConfig
			}()

			// Set the config to use our test server
			config.Config.EnterpriseApiBaseUrl = server.URL
			config.Config.EnterpriseApiToken = "test-token"

			// Call the function under test
			runId, err := RunSimulation("sim-123", "Test Title", "Test Description", nil, nil)

			// Verify the results
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, runId)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, runId)
			}
		})
	}
}

func TestGetRun(t *testing.T) {
	// Setup a mock server to handle the request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		assert.Equal(t, "/run", r.URL.Path)
		assert.Equal(t, "run=run-123", r.URL.RawQuery)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := GatlingRunResponse{
			Status:      3, // Injecting
			Error:       "",
			DeployStart: 1625000000000,
			DeployEnd:   1625000010000,
			InjectStart: 1625000010000,
			InjectEnd:   0, // Still running
			Assertions: []GatlingRunAssertion{
				{
					Message:     "Response time < 500ms",
					Result:      true,
					ActualValue: 250,
				},
				{
					Message:     "Error rate < 1%",
					Result:      false,
					ActualValue: 2.5,
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Save the original config and restore it after the test
	originalConfig := config.Config
	defer func() {
		config.Config = originalConfig
	}()

	// Set the config to use our test server
	config.Config.EnterpriseApiBaseUrl = server.URL
	config.Config.EnterpriseApiToken = "test-token"

	// Call the function under test
	run, err := GetRun("run-123")

	// Verify the results
	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, 3, run.Status) // Injecting
	assert.Empty(t, run.Error)
	assert.Equal(t, int64(1625000000000), run.DeployStart)
	assert.Equal(t, int64(1625000010000), run.DeployEnd)
	assert.Equal(t, int64(1625000010000), run.InjectStart)
	assert.Equal(t, int64(0), run.InjectEnd)

	// Check assertions
	require.Len(t, run.Assertions, 2)
	assert.Equal(t, "Response time < 500ms", run.Assertions[0].Message)
	assert.True(t, run.Assertions[0].Result)
	assert.Equal(t, 250.0, run.Assertions[0].ActualValue)
	assert.Equal(t, "Error rate < 1%", run.Assertions[1].Message)
	assert.False(t, run.Assertions[1].Result)
	assert.Equal(t, 2.5, run.Assertions[1].ActualValue)
}

func TestGetRun_Error(t *testing.T) {
	// Test cases for different error scenarios
	testCases := []struct {
		name           string
		serverStatus   int
		serverResponse string
		expectError    bool
	}{
		{
			name:           "API returns error status code",
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `{"error": "Internal server error"}`,
			expectError:    true,
		},
		{
			name:           "API returns invalid JSON",
			serverStatus:   http.StatusOK,
			serverResponse: `invalid json`,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup a mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.serverStatus)
				fmt.Fprint(w, tc.serverResponse)
			}))
			defer server.Close()

			// Save the original config and restore it after the test
			originalConfig := config.Config
			defer func() {
				config.Config = originalConfig
			}()

			// Set the config to use our test server
			config.Config.EnterpriseApiBaseUrl = server.URL
			config.Config.EnterpriseApiToken = "test-token"

			// Call the function under test
			run, err := GetRun("run-123")

			// Verify the results
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, run)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, run)
			}
		})
	}
}

func TestStopRun(t *testing.T) {
	// Setup a mock server to handle the request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		assert.Equal(t, "/simulations/abort", r.URL.Path)
		assert.Equal(t, "run=run-123", r.URL.RawQuery)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		// Return a success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"success": true}`)
	}))
	defer server.Close()

	// Save the original config and restore it after the test
	originalConfig := config.Config
	defer func() {
		config.Config = originalConfig
	}()

	// Set the config to use our test server
	config.Config.EnterpriseApiBaseUrl = server.URL
	config.Config.EnterpriseApiToken = "test-token"

	// Call the function under test
	err := StopRun("run-123")

	// Verify the results
	require.NoError(t, err)
}

func TestStopRun_Error(t *testing.T) {
	// Test cases for different error scenarios
	testCases := []struct {
		name           string
		serverStatus   int
		serverResponse string
		expectError    bool
	}{
		{
			name:           "API returns error status code",
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `{"error": "Internal server error"}`,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup a mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.serverStatus)
				fmt.Fprint(w, tc.serverResponse)
			}))
			defer server.Close()

			// Save the original config and restore it after the test
			originalConfig := config.Config
			defer func() {
				config.Config = originalConfig
			}()

			// Set the config to use our test server
			config.Config.EnterpriseApiBaseUrl = server.URL
			config.Config.EnterpriseApiToken = "test-token"

			// Call the function under test
			err := StopRun("run-123")

			// Verify the results
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetClient(t *testing.T) {
	// Save the original config and restore it after the test
	originalConfig := config.Config
	defer func() {
		config.Config = originalConfig
	}()

	// Test with InsecureSkipVerify=false
	config.Config.InsecureSkipVerify = false
	client := getClient()
	assert.NotNil(t, client)

	// Test with InsecureSkipVerify=true
	config.Config.InsecureSkipVerify = true
	client = getClient()
	assert.NotNil(t, client)

	// We can't easily test the TLS config directly, but we can verify
	// the client is returned in both cases
}
