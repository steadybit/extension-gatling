// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestWithMinikube(t *testing.T) {
	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-gatling",
		Port: 8087,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{"--set", "logging.level=debug"}
		},
	}

	mOpts := e2e.DefaultMiniKubeOpts
	mOpts.Runtimes = []e2e.Runtime{e2e.RuntimeDocker}

	e2e.WithMinikube(t, mOpts, &extFactory, []e2e.WithMinikubeTestCase{
		{
			Name: "run gatling",
			Test: testRunGatling,
		},
	})
}

func testRunGatling(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	config := struct{}{}
	files := []e2e.File{
		{
			ParameterName: "file",
			FileName:      "basic.scala",
			Content: []byte("" +
				"package com.steadybit.gatling\n\nimport scala.concurrent.duration._\n\n\nimport io.gatling.core.Predef._\nimport io.gatling.http.Predef._\n\nclass BasicScalaSimulation extends Simulation {\n\n  val httpProtocol = http\n    .baseUrl(\"http://demo.steadybit.io/products\")\n    .acceptHeader(\"application/json\")\n    .acceptEncodingHeader(\"gzip, deflate\")\n\n  val scn = scenario(\"ExampleScalaSimulation\")\n    .exec(http(\"fetch-products\")\n      .get(\"/\"))\n    .pause(5)\n\n  setUp(\n    scn.inject(atOnceUsers(1))\n  ).protocols(httpProtocol)\n}"),
		},
	}
	exec, err := e.RunActionWithFiles("com.github.steadybit.extension_gatling.run", nil, config, nil, files)
	require.NoError(t, err)
	e2e.AssertProcessRunningInContainer(t, m, e.Pod, "extension", "gatling.sh", true)
	e2e.AssertLogContainsWithTimeout(t, m, e.Pod, "Simulation com.steadybit.gatling.BasicScalaSimulation started", 90*time.Second)
	require.NoError(t, exec.Cancel())
}
