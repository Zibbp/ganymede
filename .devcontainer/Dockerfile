ARG TWITCHDOWNLOADER_VERSION="1.56.5"
ARG YT_DLP_VERSION="2026.07.04"
ARG FFMPEG_VERSION="9.0"

FROM mcr.microsoft.com/devcontainers/go:2-1.26

ARG TWITCHDOWNLOADER_VERSION
ARG YT_DLP_VERSION
ARG FFMPEG_VERSION

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
  && apt-get -y install --no-install-recommends python3-pip xz-utils curl ca-certificates fontconfig \
  && apt-get clean -y && rm -rf /var/lib/apt/lists/*

WORKDIR /tmp

# Install ffmpeg/ffprobe (static build, same version as main Dockerfile)
RUN set -eux; \
    ARCH="$(uname -m)"; \
    case "$ARCH" in \
      x86_64) FFMPEG_ARCH="linux64" ;; \
      aarch64) FFMPEG_ARCH="linuxarm64" ;; \
      *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;; \
    esac; \
    FFMPEG_TAR="ffmpeg-n${FFMPEG_VERSION}-latest-${FFMPEG_ARCH}-gpl-${FFMPEG_VERSION}.tar.xz"; \
    echo "Downloading ${FFMPEG_TAR} (FFmpeg ${FFMPEG_VERSION} latest, ${ARCH})"; \
    curl -fSL "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/${FFMPEG_TAR}" -o ffmpeg.tar.xz; \
    curl -fSL "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/checksums.sha256" -o checksums.sha256; \
    EXPECTED_HASH="$(grep -F " ${FFMPEG_TAR}" checksums.sha256 | awk '{print $1}')"; \
    if [ -z "$EXPECTED_HASH" ]; then echo "checksum entry not found for ${FFMPEG_TAR}" >&2; exit 1; fi; \
    echo "${EXPECTED_HASH}  ffmpeg.tar.xz" | sha256sum -c -; \
    mkdir -p /tmp/ffmpeg-extract; \
    tar -xJf ffmpeg.tar.xz -C /tmp/ffmpeg-extract --strip-components=1; \
    cp /tmp/ffmpeg-extract/bin/ffmpeg /usr/local/bin/ffmpeg; \
    cp /tmp/ffmpeg-extract/bin/ffprobe /usr/local/bin/ffprobe; \
    chmod +x /usr/local/bin/ffmpeg /usr/local/bin/ffprobe; \
    rm -rf /tmp/ffmpeg.tar.xz /tmp/checksums.sha256 /tmp/ffmpeg-extract; \
    ffmpeg -version; \
    ffprobe -version

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

