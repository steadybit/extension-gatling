/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package extgatling

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/steadybit/extension-gatling/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscovery_Describe(t *testing.T) {
	description := (&gatlingLocationDiscovery{}).Describe()

	assert.Equal(t, targetType, description.Id)
	require.NotNil(t, description.Discover.CallInterval)
	assert.Equal(t, "300s", *description.Discover.CallInterval)
}

func TestDiscovery_DescribeTarget(t *testing.T) {
	description := (&gatlingLocationDiscovery{}).DescribeTarget()

	assert.Equal(t, targetType, description.Id)
	assert.Equal(t, "Gatling Location", description.Label.One)
	assert.Equal(t, "execution locations", *description.Category)
	assert.Len(t, description.Table.Columns, 4)
	assert.Equal(t, "k8s.cluster-name", description.Table.OrderBy[0].Attribute)
}

func TestDiscovery_DiscoverTargets(t *testing.T) {
	originalConfig := config.Config
	t.Cleanup(func() { config.Config = originalConfig })

	t.Run("kubernetes pod", func(t *testing.T) {
		config.Config.KubernetesNamespace = "steadybit-agent"
		config.Config.KubernetesPodName = "extension-gatling-0"
		config.Config.KubernetesNodeName = "node-1"
		config.Config.KubernetesClusterName = "dev-cluster"

		targets, err := (&gatlingLocationDiscovery{}).DiscoverTargets(context.Background())

		require.NoError(t, err)
		require.Len(t, targets, 1)
		assert.Equal(t, "steadybit-agent-extension-gatling-0", targets[0].Id)
		assert.Equal(t, "steadybit-agent/extension-gatling-0", targets[0].Label)
		assert.Equal(t, targetType, targets[0].TargetType)
		assert.Equal(t, map[string][]string{
			"k8s.namespace":    {"steadybit-agent"},
			"k8s.pod.name":     {"extension-gatling-0"},
			"k8s.node.name":    {"node-1"},
			"host.hostname":    {"node-1"},
			"k8s.cluster-name": {"dev-cluster"},
		}, targets[0].Attributes)
	})

	t.Run("plain process", func(t *testing.T) {
		config.Config.KubernetesNamespace = ""
		config.Config.KubernetesPodName = ""
		config.Config.KubernetesNodeName = ""
		config.Config.KubernetesClusterName = ""
		hostname, _ := os.Hostname()
		pid := os.Getpid()

		targets, err := (&gatlingLocationDiscovery{}).DiscoverTargets(context.Background())

		require.NoError(t, err)
		require.Len(t, targets, 1)
		assert.Equal(t, fmt.Sprintf("%s-%d", hostname, pid), targets[0].Id)
		assert.Equal(t, fmt.Sprintf("%s/%d", hostname, pid), targets[0].Label)
		assert.Equal(t, map[string][]string{
			"host.hostname": {hostname},
			"process.pid":   {fmt.Sprintf("%d", pid)},
		}, targets[0].Attributes)
	})
}
