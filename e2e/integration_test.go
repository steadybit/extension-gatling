// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/client"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWithMinikube(t *testing.T) {
	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-gatling",
		Port: 8087,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{
				"--set", "logging.level=debug",
			}
		},
	}

	e2e.WithDefaultMinikube(t, &extFactory, []e2e.WithMinikubeTestCase{
		{
			Name: "run gatling with scala",
			Test: testRunGatlingWithScala,
		},
		{
			Name: "run gatling with java",
			Test: testRunGatlingWithJava,
		},
		{
			Name: "run gatling with java zip",
			Test: testRunGatlingWithJavaZip,
		},
		{
			Name: "run gatling with kotlin",
			Test: testRunGatlingWithKotlin,
		},
	})
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
	e2e.AssertLogContainsWithTimeout(t, m, e.Pod, "Simulation BasicSimulation started", 120*time.Second)
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
