ARG TWITCHDOWNLOADER_VERSION="1.54.9"

# Build stage
FROM --platform=$BUILDPLATFORM golang:1.22-bookworm AS build
WORKDIR /app
COPY . .
RUN make build_server build_worker

# Tools stage
FROM --platform=$BUILDPLATFORM debian:bookworm-slim AS tools
WORKDIR /tmp
RUN apt-get update && apt-get install -y --no-install-recommends \
unzip git ca-certificates curl \
&& rm -rf /var/lib/apt/lists/*

# Download TwitchDownloader for the correct platform
ARG TWITCHDOWNLOADER_VERSION
ENV TWITCHDOWNLOADER_URL=https://github.com/lay295/TwitchDownloader/releases/download/${TWITCHDOWNLOADER_VERSION}/TwitchDownloaderCLI-${TWITCHDOWNLOADER_VERSION}-Linux
RUN if [ "$BUILDPLATFORM" = "arm64" ]; then \
        export TWITCHDOWNLOADER_URL=${TWITCHDOWNLOADER_URL}Arm; \
    fi && \
    export TWITCHDOWNLOADER_URL=${TWITCHDOWNLOADER_URL}-x64.zip && \
    echo "Download URL: $TWITCHDOWNLOADER_URL" && \
    curl -L $TWITCHDOWNLOADER_URL -o twitchdownloader.zip && \
    unzip twitchdownloader.zip && \
    rm twitchdownloader.zip
RUN git clone --depth 1 https://github.com/xenova/chat-downloader.git

# Production stage
FROM --platform=$BUILDPLATFORM debian:bookworm-slim
WORKDIR /opt/app

# Install dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip fontconfig ffmpeg tzdata procps \
    fonts-noto-core fonts-noto-cjk fonts-noto-extra fonts-inter \
    curl \
    && rm -rf /var/lib/apt/lists/* \
    && ln -sf python3 /usr/bin/python

# Install pip packages
RUN pip3 install --no-cache-dir --upgrade pip streamlink --break-system-packages

# Install gosu
RUN curl -O https://github.com/tianon/gosu/releases/latest/download/gosu-$(dpkg --print-architecture | awk -F- '{ print $NF }') \
    && chmod 0755 gosu-$(dpkg --print-architecture | awk -F- '{ print $NF }') \
    && mv gosu-$(dpkg --print-architecture | awk -F- '{ print $NF }') /usr/local/bin/gosu

# Setup user
RUN useradd -u 911 -d /data abc && usermod -a -G users abc

# Copy and install chat-downloader
COPY --from=tools /tmp/chat-downloader /tmp/chat-downloader
RUN cd /tmp/chat-downloader && python3 setup.py install && cd .. && rm -rf chat-downloader

# Setup fonts
RUN chmod 644 /usr/share/fonts/* && chmod -R a+rX /usr/share/fonts

# Copy TwitchDownloaderCLI
COPY --from=tools /tmp/TwitchDownloaderCLI /usr/local/bin/
RUN chmod +x /usr/local/bin/TwitchDownloaderCLI

# Copy application files
COPY --from=build /app/ganymede-api .
COPY --from=build /app/ganymede-worker .

# Setup entrypoint
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 4000
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
