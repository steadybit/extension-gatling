// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extgatlingenterprise

import (
	"testing"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/stretchr/testify/assert"
)

func TestParameterGroups(t *testing.T) {
	desc := NewGatlingEnterpriseRunAction().Describe()

	var order, grouped []string
	for _, p := range desc.Parameters {
		order = append(order, p.Name)
		if p.Group == nil {
			// No advanced parameter is left alone in "Additional Settings" on platforms with groups.
			assert.False(t, p.Advanced != nil && *p.Advanced, "parameter %s", p.Name)
			continue
		}
		assert.Equal(t, groupSimulationInputs, *p.Group)
		// Platforms without groups keep showing them in "Additional Settings".
		assert.True(t, p.Advanced != nil && *p.Advanced, "parameter %s", p.Name)
		grouped = append(grouped, p.Name)
	}
	assert.Equal(t, []string{"duration", "systemProperties", "environmentVariables"}, order)
	assert.Equal(t, []string{"systemProperties", "environmentVariables"}, grouped)
	assert.Equal(t, []action_kit_api.ParameterGroup{simulationInputsGroup}, *desc.ParameterGroups)
}
