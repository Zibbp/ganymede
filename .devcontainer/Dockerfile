ARG TWITCHDOWNLOADER_VERSION="1.56.5"
ARG YT_DLP_VERSION="2026.07.04"
ARG FFMPEG_VERSION="9.0"
ARG FFMPEG_RELEASE="autobuild-2026-09-10-15-31"
ARG FFMPEG_BUILD="n9.0.1-27-g9b0578816c"
ARG FFMPEG_SHA256_LINUX64="70b162b63517038ff18c8a50f4ea11f919a071f945c5881e87b0b2cd7173f8c7"
ARG FFMPEG_SHA256_LINUXARM64="2de63f615df3a46ac26921ac0314f501c3536aaf9fd5197ffe03dd17cd01a404"

FROM mcr.microsoft.com/devcontainers/go:2-1.26

ARG TWITCHDOWNLOADER_VERSION
ARG YT_DLP_VERSION
ARG FFMPEG_VERSION
ARG FFMPEG_RELEASE
ARG FFMPEG_BUILD
ARG FFMPEG_SHA256_LINUX64
ARG FFMPEG_SHA256_LINUXARM64

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
  && apt-get -y install --no-install-recommends python3-pip xz-utils curl ca-certificates fontconfig \
  && apt-get clean -y && rm -rf /var/lib/apt/lists/*

WORKDIR /tmp

# Install ffmpeg/ffprobe (static build, same version as main Dockerfile)
RUN set -eux; \
    ARCH="$(uname -m)"; \
    case "$ARCH" in \
      x86_64) FFMPEG_ARCH="linux64"; EXPECTED_HASH="${FFMPEG_SHA256_LINUX64}" ;; \
      aarch64) FFMPEG_ARCH="linuxarm64"; EXPECTED_HASH="${FFMPEG_SHA256_LINUXARM64}" ;; \
      *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;; \
    esac; \
    FFMPEG_TAR="ffmpeg-${FFMPEG_BUILD}-${FFMPEG_ARCH}-gpl-${FFMPEG_VERSION}.tar.xz"; \
    echo "Downloading ${FFMPEG_TAR} (FFmpeg ${FFMPEG_VERSION}, ${ARCH}) from ${FFMPEG_RELEASE}"; \
    curl -fSL "https://github.com/BtbN/FFmpeg-Builds/releases/download/${FFMPEG_RELEASE}/${FFMPEG_TAR}" -o ffmpeg.tar.xz; \
    if [ -z "$EXPECTED_HASH" ]; then echo "checksum entry not found for ${FFMPEG_TAR}" >&2; exit 1; fi; \
    echo "${EXPECTED_HASH}  ffmpeg.tar.xz" | sha256sum -c -; \
    mkdir -p /tmp/ffmpeg-extract; \
    tar -xJf ffmpeg.tar.xz -C /tmp/ffmpeg-extract --strip-components=1; \
    cp /tmp/ffmpeg-extract/bin/ffmpeg /usr/local/bin/ffmpeg; \
    cp /tmp/ffmpeg-extract/bin/ffprobe /usr/local/bin/ffprobe; \
    chmod +x /usr/local/bin/ffmpeg /usr/local/bin/ffprobe; \
    rm -rf /tmp/ffmpeg.tar.xz /tmp/ffmpeg-extract; \
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

