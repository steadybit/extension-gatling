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

	"github.com/steadybit/extension-gatling/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withSimulations(t *testing.T, simulations []GatlingSimulation) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/simulations" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(simulations)
	}))
	t.Cleanup(server.Close)

	originalConfig := config.Config
	t.Cleanup(func() { config.Config = originalConfig })
	config.Config.EnterpriseApiBaseUrl = server.URL
	config.Config.EnterpriseApiToken = "test-token"
}

// A valid interval is not tested here: the cached discovery keeps refreshing in the background, reading
// config.Config concurrently with every later test that modifies it.
func TestNewDiscoveryWithInvalidInterval(t *testing.T) {
	originalConfig := config.Config
	t.Cleanup(func() { config.Config = originalConfig })
	config.Config.EnterpriseSimulationsDiscoveryInterval = "every now and then"

	assert.Nil(t, NewDiscovery())
}

func TestDiscovery_Describe(t *testing.T) {
	description := (&gatlingEnterpriseSimulationDiscovery{}).Describe()

	assert.Equal(t, targetType, description.Id)
	require.NotNil(t, description.Discover.CallInterval)
	assert.Equal(t, "1m", *description.Discover.CallInterval)
}

func TestDiscovery_DescribeTarget(t *testing.T) {
	description := (&gatlingEnterpriseSimulationDiscovery{}).DescribeTarget()

	assert.Equal(t, targetType, description.Id)
	assert.Equal(t, "Gatling Enterprise Simulation", description.Label.One)
	assert.Equal(t, "Gatling", *description.Category)
	assert.Len(t, description.Table.Columns, 2)
	assert.Equal(t, "gatling.simulation.name", description.Table.OrderBy[0].Attribute)
}

func TestDiscovery_DescribeAttributes(t *testing.T) {
	attributes := (&gatlingEnterpriseSimulationDiscovery{}).DescribeAttributes()

	require.Len(t, attributes, 2)
	assert.Equal(t, "gatling.simulation.name", attributes[0].Attribute)
	assert.Equal(t, "gatling.simulation.class", attributes[1].Attribute)
}

func TestDiscovery_DiscoverTargets(t *testing.T) {
	withSimulations(t, []GatlingSimulation{
		{Id: "sim-1", Name: "Checkout", ClassName: "CheckoutSimulation", TeamId: "team-1", Build: GatlingSimulationBuild{PkgId: "pkg-1"}},
		{Id: "sim-2", Name: "Search", ClassName: "SearchSimulation", TeamId: "team-2"},
	})

	targets, err := (&gatlingEnterpriseSimulationDiscovery{}).DiscoverTargets(context.Background())

	require.NoError(t, err)
	require.Len(t, targets, 2)
	assert.Equal(t, "sim-1", targets[0].Id)
	assert.Equal(t, "Checkout", targets[0].Label)
	assert.Equal(t, targetType, targets[0].TargetType)
	assert.Equal(t, map[string][]string{
		"steadybit.label":               {"Checkout"},
		"gatling.simulation.id":         {"sim-1"},
		"gatling.simulation.name":       {"Checkout"},
		"gatling.simulation.class":      {"CheckoutSimulation"},
		"gatling.simulation.team.id":    {"team-1"},
		"gatling.simulation.package.id": {"pkg-1"},
	}, targets[0].Attributes)
	assert.NotContains(t, targets[1].Attributes, "gatling.simulation.package.id", "simulations without a package carry no package id")
}

func TestDiscovery_DiscoverTargetsWhenTheApiFails(t *testing.T) {
	withSimulations(t, nil)
	config.Config.EnterpriseApiBaseUrl += "/unavailable"

	targets, err := (&gatlingEnterpriseSimulationDiscovery{}).DiscoverTargets(context.Background())

	require.NoError(t, err)
	assert.Empty(t, targets)
}
