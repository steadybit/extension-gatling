<img src="./gatling-logo.png" height="130" align="right" alt="gatling logo">

# Steadybit extension-gatling

A [Steadybit](https://www.steadybit.com/) action implementation to integrate gatling load tests into Steadybit experiments.

Learn about the capabilities of this extension in our [Reliability Hub](https://hub.steadybit.com/extension/com.steadybit.extension_gatling).

## Configuration

The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

## Installation

### Using Helm in Kubernetes

```sh
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

### Using Docker

```sh
docker run \
  --rm \
  -p 8087 \
  --name steadybit-extension-gatling \
  ghcr.io/steadybit/extension-gatling:latest
```

### Linux Package

This extension is currently not available as a Linux package.

## Register the extension

Make sure to register the extension at the steadybit platform. Please refer to
the [documentation](https://docs.steadybit.com/integrate-with-steadybit/extensions/extension-installation) for more information.
