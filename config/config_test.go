/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfiguration(t *testing.T) {
	t.Setenv("STEADYBIT_EXTENSION_ENTERPRISE_ORGANIZATION_SLUG", "my-org")

	ParseConfiguration()
	ValidateConfiguration()

	assert.Equal(t, "my-org", Config.EnterpriseOrganizationSlug)
	assert.Equal(t, DefaultEnterpriseApiBaseUrl, Config.EnterpriseApiBaseUrl)
}
