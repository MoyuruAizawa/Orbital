FROM --platform=linux/amd64 ubuntu:24.04

ARG ANDROID_CMDLINE_TOOLS_VERSION=14742923
ENV ANDROID_SDK_ROOT=/opt/android-sdk
ENV ANDROID_HOME=/opt/android-sdk
ENV DEBIAN_FRONTEND=noninteractive
ENV GRADLE_OPTS=-Dorg.gradle.vfs.watch=false

ARG RUNNER_VERSION=2.333.1
ENV RUNNER_ALLOW_RUNASROOT=1
ENV RUNNER_WORKDIR="_work"

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        openjdk-21-jdk \
        git \
        curl \
        unzip \
        jq \
        tar \
        sudo \
        ca-certificates \
        docker.io \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p "${ANDROID_SDK_ROOT}/cmdline-tools" \
    && cd /tmp \
    && curl -L --fail --show-error -o cmdline-tools.zip \
        "https://dl.google.com/android/repository/commandlinetools-linux-${ANDROID_CMDLINE_TOOLS_VERSION}_latest.zip" \
    && unzip -q cmdline-tools.zip -d "${ANDROID_SDK_ROOT}/cmdline-tools" \
    && mv "${ANDROID_SDK_ROOT}/cmdline-tools/cmdline-tools" "${ANDROID_SDK_ROOT}/cmdline-tools/latest" \
    && rm -f cmdline-tools.zip

RUN mkdir -p /root/.android \
    && touch /root/.android/repositories.cfg

RUN export JAVA_HOME="$(dirname "$(dirname "$(readlink -f "$(which javac)")")")" \
    && export PATH="$PATH:$JAVA_HOME/bin:${ANDROID_SDK_ROOT}/cmdline-tools/latest/bin:${ANDROID_SDK_ROOT}/platform-tools" \
    && yes | sdkmanager --sdk_root="${ANDROID_SDK_ROOT}" --licenses

RUN export JAVA_HOME="$(dirname "$(dirname "$(readlink -f "$(which javac)")")")" \
    && export PATH="$PATH:$JAVA_HOME/bin:${ANDROID_SDK_ROOT}/cmdline-tools/latest/bin:${ANDROID_SDK_ROOT}/platform-tools" \
    && sdkmanager --sdk_root="${ANDROID_SDK_ROOT}" \
        "platform-tools" \
        "platforms;android-35" \
        "platforms;android-36" \
        "build-tools;35.0.1" \
        "build-tools;36.1.0"

WORKDIR /actions-runner
RUN curl -L --fail --show-error -o actions-runner.tar.gz -L https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz \
    && tar xzf actions-runner.tar.gz \
    && rm actions-runner.tar.gz \
    && ./bin/installdependencies.sh

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
