package e2e

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
)

const simulationId = "27161d73-816c-480d-ae3d-c8bacd57e661"

func createGatlingEnterpriseMock() *httptest.Server {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen: %v", err))
	}
	server := httptest.Server{
		Listener: listener,
		Config: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Info().Str("path", r.URL.Path).Str("method", r.Method).Str("query", r.URL.RawQuery).Msg("Request received")
			if strings.Contains(r.URL.Path, "/simulations/start") {
				w.WriteHeader(http.StatusOK)
				w.Write(startResponse())
			} else if strings.Contains(r.URL.Path, "/run") {
				w.WriteHeader(http.StatusOK)
				w.Write(getRun())
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
		})},
	}
	server.Start()
	log.Info().Str("url", server.URL).Msg("Started Mock-Server")
	return &server
}

func startResponse() []byte {
	return []byte(`{
  "className": "example.SteadybitExampleSimulation",
  "runId": "0b1c0811-f9a3-45ef-af5a-ac4fb2030cba",
  "reportsPath": "/o/steadybit/simulations/27161d73-816c-480d-ae3d-c8bacd57e661/runs/0b1c0811-f9a3-45ef-af5a-ac4fb2030cba",
  "runsPath": "/o/steadybit/simulations/27161d73-816c-480d-ae3d-c8bacd57e661/runs"
}`)
}

func getRun() []byte {
	log.Info().Msg("Return collection")
	return []byte(`{
  "incrementalId": 1,
  "runId": "0b1c0811-f9a3-45ef-af5a-ac4fb2030cba",
  "buildStart": 1747147517422,
  "buildEnd": 1747147532949,
  "deployStart": 0,
  "deployEnd": 0,
  "injectStart": 0,
  "injectEnd": 0,
  "status": 7,
  "runSnapshot": {
    "simulationName": "10 times google with assertions",
    "systemProperties": {},
    "environmentVariables": {},
    "useLocationWeights": false,
    "locationSnapshots": [
      {
        "locationId": "4a399023-d443-3a58-864f-3919760df78b",
        "locationName": "Europe - Paris",
        "size": 1,
        "weight": 0,
        "dedicatedIps": []
      }
    ],
    "stopCriteria": [],
    "simulationClass": "example.SteadybitExampleSimulation",
    "trigger": {
      "type": "apiToken",
      "tokenId": "abecefa7-49af-4acd-971a-cd70e752cbb6",
      "name": "extension-gatling"
    },
    "organizationId": "7fb1c2f2-7052-41bc-b632-e71702a009da"
  },
  "comments": {
    "title": "My run title",
    "description": "My run description"
  },
  "assertions": [],
  "scenario": "",
  "request": "%1A",
  "group": ""
}`)
}
