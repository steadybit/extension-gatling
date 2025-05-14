// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package extgatlingenterprise

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-gatling/config"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"time"
)

type gatlingEnterpriseSimulationDiscovery struct{}

var (
	_ discovery_kit_sdk.TargetDescriber = (*gatlingEnterpriseSimulationDiscovery)(nil)
)

func NewDiscovery() discovery_kit_sdk.TargetDiscovery {
	discovery := &gatlingEnterpriseSimulationDiscovery{}
	interval, err := time.ParseDuration(config.Config.EnterpriseSimulationsDiscoveryInterval)
	if err != nil {
		log.Error().Msgf("Failed to parse discovery interval: %s", err)
		return nil
	}
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsInterval(context.Background(), interval),
	)
}

func (e *gatlingEnterpriseSimulationDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: targetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func (e *gatlingEnterpriseSimulationDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       targetType,
		Label:    discovery_kit_api.PluralLabel{One: "Gatling Enterprise Simulation", Other: "Gatling Enterprise Simulations"},
		Category: extutil.Ptr("Gatling"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(targetIcon),

		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "gatling.simulation.name"},
				{Attribute: "gatling.simulation.class"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "gatling.simulation.name",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *gatlingEnterpriseSimulationDiscovery) DescribeAttributes() []discovery_kit_api.AttributeDescription {
	return []discovery_kit_api.AttributeDescription{
		{
			Attribute: "gatling.simulation.name",
			Label: discovery_kit_api.PluralLabel{
				One:   "Gatling Simulation Name",
				Other: "Gatling Simulation Names",
			},
		},
		{
			Attribute: "gatling.simulation.class",
			Label: discovery_kit_api.PluralLabel{
				One:   "Gatling Simulation Class",
				Other: "Gatling Simulation Classes",
			},
		},
	}
}

func (e *gatlingEnterpriseSimulationDiscovery) DiscoverTargets(_ context.Context) ([]discovery_kit_api.Target, error) {
	simulations := GetSimulations()
	targets := make([]discovery_kit_api.Target, len(simulations))

	for i, simulation := range simulations {
		attributes := map[string][]string{
			"steadybit.label":            {simulation.Name},
			"gatling.simulation.id":      {simulation.Id},
			"gatling.simulation.name":    {simulation.Name},
			"gatling.simulation.class":   {simulation.ClassName},
			"gatling.simulation.team.id": {simulation.TeamId},
		}
		if simulation.Build.PkgId != "" {
			attributes["gatling.simulation.package.id"] = []string{simulation.Build.PkgId}
		}
		targets[i] = discovery_kit_api.Target{
			Id:         simulation.Id,
			TargetType: targetType,
			Label:      simulation.Name,
			Attributes: attributes,
		}
	}
	return targets, nil
}
