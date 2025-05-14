// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/client"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"github.com/steadybit/discovery-kit/go/discovery_kit_test/validate"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWithMinikube(t *testing.T) {
	extlogging.InitZeroLog()
	server := createGatlingEnterpriseMock()
	defer server.Close()
	split := strings.SplitAfter(server.URL, ":")
	port := split[len(split)-1]

	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-gatling",
		Port: 8087,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{
				"--set", "logging.level=debug",
				"--set", "gatling.enterpriseApiToken=testToken",
				"--set", "extraEnv[0].name=STEADYBIT_EXTENSION_ENTERPRISE_API_BASE_URL",
				"--set", fmt.Sprintf("extraEnv[0].value=%s:%s", "http://host.minikube.internal", port),
			}
		},
	}

	e2e.WithDefaultMinikube(t, &extFactory, []e2e.WithMinikubeTestCase{
		{
			Name: "validate discovery",
			Test: validateDiscovery,
		},
		//{
		//	Name: "run gatling with scala",
		//	Test: testRunGatlingWithScala,
		//},
		//{
		//	Name: "run gatling with java",
		//	Test: testRunGatlingWithJava,
		//},
		//{
		//	Name: "run gatling with java zip",
		//	Test: testRunGatlingWithJavaZip,
		//},
		//{
		//	Name: "run gatling with kotlin",
		//	Test: testRunGatlingWithKotlin,
		//},
		{
			Name: "run gatling enterprise simulation",
			Test: testRunGatlingEnterpriseSimulation,
		},
	})
}

func validateDiscovery(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	assert.NoError(t, validate.ValidateEndpointReferences("/", e.Client))
}

func readFileContent(t *testing.T, filePath string) string {
	// Get the absolute path to ensure we can find the file
	absPath, err := filepath.Abs(filePath)
	require.NoError(t, err, "failed to get absolute path for: "+filePath)
	content, err := os.ReadFile(absPath)
	require.NoError(t, err, "failed to read file: "+absPath)
	return string(content)
}

func testRunGatling(t *testing.T, m *e2e.Minikube, e *e2e.Extension, fileName string, filePath string) {
	config := struct{}{}
	context := &action_kit_api.ExecutionContext{ExperimentKey: extutil.Ptr("ADM-1"), ExecutionId: extutil.Ptr(4711)}
	content := readFileContent(t, filePath)
	files := []client.File{
		{
			ParameterName: "file",
			FileName:      fileName,
			Content:       []byte(content),
		},
	}
	exec, err := e.RunActionWithFiles("com.steadybit.extension_gatling.run", nil, config, context, files)
	require.NoError(t, err)
	e2e.AssertLogContainsWithTimeout(t, m, e.Pod, "Simulation BasicSimulation started", 60*time.Second)
	e2e.AssertLogContainsWithTimeout(t, m, e.Pod, "BUILD SUCCESS", 60*time.Second)
	require.NoError(t, exec.Cancel())
}

func testRunGatlingWithScala(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	testRunGatling(t, m, e, "BasicSimulation.scala", "../examples/BasicSimulation.scala")
}

func testRunGatlingWithJava(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	testRunGatling(t, m, e, "BasicSimulation.java", "../examples/BasicSimulation.java")
}

func testRunGatlingWithJavaZip(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	testRunGatling(t, m, e, "BasicSimulation.zip", "../examples/BasicSimulation.zip")
}

func testRunGatlingWithKotlin(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	testRunGatling(t, m, e, "BasicSimulation.kt", "../examples/BasicSimulation.kt")
}

func testRunGatlingEnterpriseSimulation(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	config := struct{}{}

	target := action_kit_api.Target{
		Attributes: map[string][]string{
			"gatling.simulation.id": {simulationId},
		},
	}
	context := &action_kit_api.ExecutionContext{ExperimentKey: extutil.Ptr("ADM-1"), ExecutionId: extutil.Ptr(4711)}

	exec, err := e.RunAction("com.steadybit.extension_gatling.enterprise.simulation.run", &target, config, context)
	require.NoError(t, err)

	e2e.AssertLogContainsWithTimeout(t, m, e.Pod, "Simulation ended", 60*time.Second)
	require.NoError(t, exec.Cancel())
}
