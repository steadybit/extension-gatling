# syntax=docker/dockerfile:1

##
## Build
##
FROM --platform=$BUILDPLATFORM golang:1.25-trixie AS build

ARG TARGETOS
ARG TARGETARCH
ARG NAME
ARG VERSION
ARG REVISION
ARG ADDITIONAL_BUILD_PARAMS
ARG SKIP_LICENSES_REPORT=false
ARG VERSION=unknown
ARG REVISION=unknown

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
FROM azul/zulu-openjdk-debian:25

ARG VERSION=unknown
ARG REVISION=unknown

LABEL "steadybit.com.discovery-disabled"="true"
LABEL "version"="${VERSION}"
LABEL "revision"="${REVISION}"
RUN echo "$VERSION" > /version.txt && echo "$REVISION" > /revision.txt

RUN apt-get -qq update && \
    apt-get -qq -y upgrade && \
    apt-get -qq -y --no-install-recommends install procps unzip zip wget tar && \
    rm -rf /var/lib/apt/lists/*

# Install Maven
ENV MAVEN_VERSION=3.9.12
ENV MAVEN_BASE_URL=https://archive.apache.org/dist/maven/maven-3/${MAVEN_VERSION}/binaries
ENV MAVEN_FILENAME=apache-maven-${MAVEN_VERSION}-bin.tar.gz
RUN apt-get update && apt-get install -y wget tar && \
    wget ${MAVEN_BASE_URL}/${MAVEN_FILENAME} --max-redirect=0 -O /tmp/${MAVEN_FILENAME} && \
    tar -xzf /tmp/${MAVEN_FILENAME} -C /opt/ && \
    rm -rf /var/lib/apt/lists/* /tmp/${MAVEN_FILENAME} && \
    rm -rf /opt/apache-maven-${MAVEN_VERSION}/bin/mvnDebug \
           /opt/apache-maven-${MAVEN_VERSION}/bin/mvnyjp \
           /opt/apache-maven-${MAVEN_VERSION}/man \
           /opt/apache-maven-${MAVEN_VERSION}/lib/ext \
           /opt/apache-maven-${MAVEN_VERSION}/lib/jansi-native \
           /opt/apache-maven-${MAVEN_VERSION}/NOTICE \
           /opt/apache-maven-${MAVEN_VERSION}/README.txt \
           /opt/apache-maven-${MAVEN_VERSION}/doc \
           /opt/apache-maven-${MAVEN_VERSION}/src.zip && \
    ln -s /opt/apache-maven-${MAVEN_VERSION}/bin/mvn /usr/bin/mvn
ENV MAVEN_HOME=/opt/apache-maven-${MAVEN_VERSION}
ENV PATH="${MAVEN_HOME}/bin:${PATH}"

COPY gatling-maven-scaffold /gatling-maven-scaffold
COPY examples/BasicSimulation.java /gatling-maven-scaffold/src/test/java/BasicSimulation.java
COPY examples/BasicSimulation.kt /gatling-maven-scaffold/src/test/kotlin/BasicSimulation.kt
COPY examples/BasicSimulation.scala /gatling-maven-scaffold/src/test/scala/BasicSimulation.scala

# Setup user
ARG USERNAME=steadybit
ARG USER_UID=10000
ARG USER_GID=$USER_UID
RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && chown -R steadybit /gatling-maven-scaffold

USER $USER_UID

RUN mkdir -p /tmp/.java/.systemPrefs /tmp/.java/.userPrefs /tmp/.kotlin && \
    chmod -R 755 /tmp/.java /tmp/.kotlin

ENV KOTLIN_USER_HOME=/tmp/.kotlin
ENV JAVA_OPTS="-Djava.util.prefs.systemRoot=/tmp/.java -Djava.util.prefs.userRoot=/tmp/.java/.userPrefs -Dsteadybit.agent.disable-jvm-attachment"
ENV MAVEN_OPTS="-Djava.util.prefs.systemRoot=/tmp/.java -Djava.util.prefs.userRoot=/tmp/.java/.userPrefs -Dsteadybit.agent.disable-jvm-attachment"

# Run a simple test to pre-load all required dependencies
RUN cd /gatling-maven-scaffold && \
    mvn integration-test && \
    rm -rf /gatling-maven-scaffold/target && \
    rm -rf /gatling-maven-scaffold/src/test/java && \
    mvn integration-test -Pkotlin && \
    rm -rf /gatling-maven-scaffold/target && \
    rm -rf /gatling-maven-scaffold/src/test/kotlin && \
    mvn integration-test -Pscala && \
    rm -rf /gatling-maven-scaffold/target && \
    rm -rf /gatling-maven-scaffold/src/test/scala

WORKDIR /

COPY --from=build /app/extension /extension
COPY --from=build /app/licenses /licenses

EXPOSE 8087
EXPOSE 8088

ENTRYPOINT ["/extension"]
