FROM mcr.microsoft.com/devcontainers/go:1

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
  && apt-get -y install --no-install-recommends ffmpeg python3 python3-pip \
  && apt-get clean -y && rm -rf /var/lib/apt/lists/*

RUN pip3 install --no-cache --upgrade --break-system-packages pip streamlink chat-downloader

WORKDIR /tmp

RUN wget https://github.com/rsms/inter/releases/download/v4.0-beta7/Inter-4.0-beta7.zip && unzip Inter-4.0-beta7.zip && mkdir -p /usr/share/fonts/opentype/inter/ && cp /tmp/Desktop/Inter-*.otf /usr/share/fonts/opentype/inter/ && fc-cache -f -v

RUN wget https://github.com/lay295/TwitchDownloader/releases/download/1.55.0/TwitchDownloaderCLI-1.55.0-Linux-x64.zip && unzip TwitchDownloaderCLI-1.55.0-Linux-x64.zip && mv TwitchDownloaderCLI /usr/local/bin/ && chmod +x /usr/local/bin/TwitchDownloaderCLI && rm TwitchDownloaderCLI-1.55.0-Linux-x64.zip

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.50.1
