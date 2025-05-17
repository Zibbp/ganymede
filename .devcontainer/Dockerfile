FROM mcr.microsoft.com/devcontainers/go:1

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
  && apt-get -y install --no-install-recommends ffmpeg \
  && apt-get clean -y && rm -rf /var/lib/apt/lists/*

WORKDIR /tmp

RUN wget https://github.com/rsms/inter/releases/download/v4.1/Inter-4.1.zip && unzip Inter-4.1.zip && mkdir -p /usr/share/fonts/opentype/inter/ && cp /tmp/extras/otf/Inter-*.otf /usr/share/fonts/opentype/inter/ && fc-cache -f -v

RUN wget https://github.com/lay295/TwitchDownloader/releases/download/1.55.7/TwitchDownloaderCLI-1.55.7-Linux-x64.zip && unzip TwitchDownloaderCLI-1.55.7-Linux-x64.zip && mv TwitchDownloaderCLI /usr/local/bin/ && chmod +x /usr/local/bin/TwitchDownloaderCLI && rm TwitchDownloaderCLI-1.55.7-Linux-x64.zip
