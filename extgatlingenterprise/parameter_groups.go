// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extgatlingenterprise

import "github.com/steadybit/action-kit/go/action_kit_api/v2"

const groupSimulationInputs = "Simulation Inputs"

// They are passed as extraSystemProperties and extraEnvironmentVariables of the Gatling Enterprise run.
var simulationInputsGroup = action_kit_api.ParameterGroup{
	Name:        groupSimulationInputs,
	Description: "Added to the system properties and environment variables configured for the simulation in Gatling Enterprise.",
}
