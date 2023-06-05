# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.20-bullseye AS build

ARG NAME
ARG VERSION
ARG REVISION

WORKDIR /app

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-recommends build-essential
COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build \
    -ldflags="\
    -X 'github.com/steadybit/extension-kit/extbuild.ExtensionName=${NAME}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Version=${VERSION}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Revision=${REVISION}'" \
    -o ./extension

##
## Runtime
##
FROM openjdk:21-slim

ENV GATLING_VERSION 3.9.5
ENV GATLING_HOME /opt/gatling
ENV GATLING_BIN ${GATLING_HOME}/bin
ENV PATH ${GATLING_BIN}:$PATH

## Installing dependencies
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-recommends wget coreutils unzip bash curl procps

# Installing jmeter
RUN mkdir -p /opt/
ADD https://repo1.maven.org/maven2/io/gatling/highcharts/gatling-charts-highcharts-bundle/${GATLING_VERSION}/gatling-charts-highcharts-bundle-${GATLING_VERSION}-bundle.zip /tmp/
RUN cd /tmp/ \
 && unzip -d /opt gatling-charts-highcharts-bundle-${GATLING_VERSION}-bundle.zip \
 && mv /opt/gatling-charts-highcharts-bundle-${GATLING_VERSION} ${GATLING_HOME} \
 && rm gatling-charts-highcharts-bundle-${GATLING_VERSION}-bundle.zip \
 && rm --recursive --force ${GATLING_HOME}/user-files/simulations/computerdatabase \
 && rm ${GATLING_HOME}/user-files/resources/search.csv

# Setup user
ARG USERNAME=steadybit
ARG USER_UID=10000
RUN adduser --uid $USER_UID $USERNAME
RUN chown -R steadybit /opt/gatling
USER $USERNAME

WORKDIR /

COPY --from=build /app/extension /extension

EXPOSE 8087
EXPOSE 8088

ENTRYPOINT ["/extension"]
