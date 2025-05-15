<img src="./gatling-logo.png" height="130" align="right" alt="gatling logo">

# Steadybit extension-gatling

A [Steadybit](https://www.steadybit.com/) action implementation to integrate gatling load tests into Steadybit experiments.

Learn about the capabilities of this extension in our [Reliability Hub](https://hub.steadybit.com/extension/com.steadybit.extension_gatling).

## Configuration

| Environment Variable                                             | Helm value                           | Meaning                                                                                                                                                                                              | required | default                           |
|------------------------------------------------------------------|--------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|-----------------------------------|
| `STEADYBIT_EXTENSION_ENTERPRISE_API_TOKEN`                       | `gatling.enterpriseApiToken`         | If you want to use Gatling Enterprise, you can provide an API token to be able to discover and run simulations via Gatling Enterprise                                                                | no       |                                   |
| `STEADYBIT_EXTENSION_ENTERPRISE_ORGANIZATION_SLUG`               | `gatling.enterpriseOrganizationSlug` | Your organization slug for Gatling Enterprise, like `steadybit`                                                                                                                                      | no       |                                   |
| `STEADYBIT_EXTENSION_ENTERPRISE_API_BASE_URL`                    | via extraEnv variables               | The base url for Gatling Enterprise                                                                                                                                                                  | no       | https://api.gatling.io/api/public |
| `STEADYBIT_EXTENSION_ENTERPRISE_SIMULATIONS_DISCOVERY_INTERVALL` | via extraEnv variables               | Discovery Interval for simulations in Gatling Enterprise                                                                                                                                             | no       | 3h                                |
| `STEADYBIT_EXTENSION_ENABLE_LOCATION_SELECTION`                  | `enableLocationSelection`            | By default, the platform will select a random instance when executing actions from this extension. If you enable location selection, users can optionally specify the location via target selection. | no       | false                             |

The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

## Installation

### Kubernetes

Detailed information about agent and extension installation in kubernetes can also be found in
our [documentation](https://docs.steadybit.com/install-and-configure/install-agent/install-on-kubernetes).

#### Recommended (via agent helm chart)

All extensions provide a helm chart that is also integrated in the
[helm-chart](https://github.com/steadybit/helm-charts/tree/main/charts/steadybit-agent) of the agent.

You must provide additional values to activate this extension.

```
--set extension-gatling.enabled=true \
```

Additional configuration options can be found in
the [helm-chart](https://github.com/steadybit/extension-gatling/blob/main/charts/steadybit-extension-gatling/values.yaml) of the
extension.

#### Alternative (via own helm chart)

If you need more control, you can install the extension via its
dedicated [helm-chart](https://github.com/steadybit/extension-gatling/blob/main/charts/steadybit-extension-gatling).

```bash
helm repo add steadybit-extension-gatling https://steadybit.github.io/extension-gatling
helm repo update
helm upgrade steadybit-extension-gatling \
    --install \
    --wait \
    --timeout 5m0s \
    --create-namespace \
    --namespace steadybit-agent \
    steadybit-extension-gatling/steadybit-extension-gatling
```

### Linux Package

This extension is currently not available as a Linux package.

## Extension registration

Make sure that the extension is registered with the agent. In most cases this is done automatically. Please refer to
the [documentation](https://docs.steadybit.com/install-and-configure/install-agent/extension-registration) for more
information about extension registration and how to verify.

## Location Selection
When multiple Gatling extensions are deployed in different subsystems (e.g., multiple Kubernetes clusters), it can be tricky to ensure that the load test is performed from the right location when testing cluster-internal URLs or having different load testing hardware sizings.
To solve this, you can activate the location selection feature.
Once you do that, the Gatling extension discovers itself as a Gatling location.
When configuring the experiment, you can optionally define which extension's deployment should execute the loadtest.
Also, the execution locations are part of Steadybit's environment concept, so you can assign permissions for execution locations.

### Migration Guideline
Before activating the location selection feature, be sure to follow these steps:
1. The installed agent version needs to be >= 2.0.47, and - only for on-prem customers - the platform version needs to be >=2.2.2
2. Activate the location selection via environment or helm variable when deploying the latest extension version (see [configuration options](#configuration).
3. Configure every environment that should be able to run Gatling load tests by including the execution location in the environment configuration.
	 One option is to add the statement `or target via the query language.type="com.steadybit.extension_gatling.location"` to your existing query.
	 You can also filter the available execution locations down, e.g., via the clustername by using `(target.type="com.steadybit.extension_gatling.location" and k8s.cluster-name="CLUSTER-NAME")`

## Version and Revision

The version and revision of the extension:
- are printed during the startup of the extension
- are added as a Docker label to the image
- are available via the `version.txt`/`revision.txt` files in the root of the image

