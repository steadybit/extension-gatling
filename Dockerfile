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
FROM debian:bullseye-slim

ENV LANG='en_US.UTF-8' LANGUAGE='en_US:en' LC_ALL='en_US.UTF-8'

ARG ZULU_REPO_VER=1.0.0-3

RUN apt-get -qq update && \
    apt-get -qq -y --no-install-recommends install gnupg software-properties-common locales curl tzdata procps unzip zip && \
    echo "en_US.UTF-8 UTF-8" >> /etc/locale.gen && \
    locale-gen en_US.UTF-8 && \
    apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 0xB1998361219BD9C9 && \
    curl -sLO https://cdn.azul.com/zulu/bin/zulu-repo_${ZULU_REPO_VER}_all.deb && dpkg -i zulu-repo_${ZULU_REPO_VER}_all.deb && \
# add testing repository to install newer packages
    add-apt-repository "deb http://httpredir.debian.org/debian testing main" && \
    apt-get -qq update && \
    mkdir -p /usr/share/man/man1 && \
    apt-get -qq -y --no-install-recommends install zulu17-jdk/stable && \
# install updates from testing due to CVE for libxml2 < 2.9.13 && libexpat1 < 2.4.5
    apt-get -qq -y --no-install-recommends -t testing install libxml2 libexpat1 && \
    apt-get -qq -y purge gnupg software-properties-common curl && \
    apt -y autoremove && \
    rm -rf /var/lib/apt/lists/* zulu-repo_${ZULU_REPO_VER}_all.deb

ENV JAVA_HOME=/usr/lib/jvm/zulu17

ENV GATLING_VERSION 3.9.5
ENV GATLING_HOME /opt/gatling
ENV GATLING_BIN ${GATLING_HOME}/bin
ENV PATH ${GATLING_BIN}:$PATH

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

RUN mkdir -p /tmp/.java/.systemPrefs \
 && mkdir /tmp/.java/.userPrefs \
 && chmod -R 755 /tmp/.java

ENV JAVA_OPTS "-Djava.util.prefs.systemRoot=/tmp/.java -Djava.util.prefs.userRoot=/tmp/.java/.userPrefs"

WORKDIR /

COPY --from=build /app/extension /extension

EXPOSE 8087
EXPOSE 8088

ENTRYPOINT ["/extension"]
