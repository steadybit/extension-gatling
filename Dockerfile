# syntax=docker/dockerfile:1

##
## Build
##
FROM --platform=$BUILDPLATFORM golang:1.22-bullseye AS build

ARG TARGETOS TARGETARCH
ARG NAME
ARG VERSION
ARG REVISION
ARG ADDITIONAL_BUILD_PARAMS
ARG SKIP_LICENSES_REPORT=false

WORKDIR /app

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-recommends build-essential
COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="\
    -X 'github.com/steadybit/extension-kit/extbuild.ExtensionName=${NAME}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Version=${VERSION}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Revision=${REVISION}'" \
    -o ./extension \
    ${ADDITIONAL_BUILD_PARAMS}
RUN make licenses-report

##
## Runtime
##
FROM azul/zulu-openjdk-debian:17

LABEL "steadybit.com.discovery-disabled"="true"

ENV GATLING_VERSION=3.10.3
ENV GATLING_HOME=/opt/gatling
ENV GATLING_BIN=${GATLING_HOME}/bin
ENV PATH=${GATLING_BIN}:$PATH

RUN apt-get -qq update && \
    apt-get -qq -y upgrade && \
    apt-get -qq -y --no-install-recommends install procps unzip zip && \
    rm -rf /var/lib/apt/lists/*

# Installing jmeter
ADD https://repo1.maven.org/maven2/io/gatling/highcharts/gatling-charts-highcharts-bundle/${GATLING_VERSION}/gatling-charts-highcharts-bundle-${GATLING_VERSION}-bundle.zip /tmp/
RUN mkdir -p /opt/  \
 && cd /tmp/ \
 && unzip -d /opt gatling-charts-highcharts-bundle-${GATLING_VERSION}-bundle.zip \
 && mv /opt/gatling-charts-highcharts-bundle-${GATLING_VERSION} ${GATLING_HOME} \
 && rm gatling-charts-highcharts-bundle-${GATLING_VERSION}-bundle.zip \
 && rm --recursive --force ${GATLING_HOME}/user-files/simulations/computerdatabase \
 && rm ${GATLING_HOME}/user-files/resources/search.csv

# Setup user
ARG USERNAME=steadybit
ARG USER_UID=10000
ARG USER_GID=$USER_UID
RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && chown -R steadybit /opt/gatling

USER $USERNAME

RUN mkdir -p /tmp/.java/.systemPrefs /tmp/.java/.userPrefs && \
    chmod -R 755 /tmp/.java

ENV JAVA_OPTS="-Djava.util.prefs.systemRoot=/tmp/.java -Djava.util.prefs.userRoot=/tmp/.java/.userPrefs"

WORKDIR /

COPY --from=build /app/extension /extension
COPY --from=build /app/licenses /licenses

EXPOSE 8087
EXPOSE 8088

ENTRYPOINT ["/extension"]
