ARG TWITCHDOWNLOADER_VERSION="1.55.7"
ARG STREAMLINK_VERSION="7.4.0"

FROM mcr.microsoft.com/devcontainers/go:1

ARG STREAMLINK_VERSION

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
  && apt-get -y install --no-install-recommends ffmpeg python3-pip \
  && apt-get clean -y && rm -rf /var/lib/apt/lists/*

WORKDIR /tmp

RUN wget https://github.com/rsms/inter/releases/download/v4.1/Inter-4.1.zip && unzip Inter-4.1.zip && mkdir -p /usr/share/fonts/opentype/inter/ && cp /tmp/extras/otf/Inter-*.otf /usr/share/fonts/opentype/inter/ && fc-cache -f -v

ARG TWITCHDOWNLOADER_VERSION
ENV TWITCHDOWNLOADER_URL=https://github.com/lay295/TwitchDownloader/releases/download/${TWITCHDOWNLOADER_VERSION}/TwitchDownloaderCLI-${TWITCHDOWNLOADER_VERSION}-Linux-x64.zip

RUN if [ "$(uname -m)" = "aarch64" ]; then \
  TWITCHDOWNLOADER_URL=https://github.com/lay295/TwitchDownloader/releases/download/${TWITCHDOWNLOADER_VERSION}/TwitchDownloaderCLI-${TWITCHDOWNLOADER_VERSION}-LinuxArm64.zip; \
  fi && \
  echo "Download URL: $TWITCHDOWNLOADER_URL" && \
  curl -L $TWITCHDOWNLOADER_URL -o twitchdownloader.zip && \
  unzip twitchdownloader.zip && \
  rm twitchdownloader.zip && \
  mv TwitchDownloaderCLI /usr/local/bin/ && \
  chmod +x /usr/local/bin/TwitchDownloaderCLI

RUN pip install --upgrade pip streamlink==${STREAMLINK_VERSION} --break-system-packages
