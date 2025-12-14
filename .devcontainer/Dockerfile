ARG TWITCHDOWNLOADER_VERSION="1.56.2"
ARG YT_DLP_VERSION="2025.11.12"

FROM mcr.microsoft.com/devcontainers/go:1.25

ARG TWITCHDOWNLOADER_VERSION
ARG YT_DLP_VERSION

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
  && apt-get -y install --no-install-recommends ffmpeg python3-pip \
  && apt-get clean -y && rm -rf /var/lib/apt/lists/*

WORKDIR /tmp

RUN wget https://github.com/rsms/inter/releases/download/v4.1/Inter-4.1.zip && unzip Inter-4.1.zip && mkdir -p /usr/share/fonts/opentype/inter/ && cp /tmp/extras/otf/Inter-*.otf /usr/share/fonts/opentype/inter/ && fc-cache -f -v

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

RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/download/${YT_DLP_VERSION}/yt-dlp -o /usr/local/bin/yt-dlp && \
  chmod +x /usr/local/bin/yt-dlp

WORKDIR /workspace

