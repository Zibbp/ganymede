ARG TWITCHDOWNLOADER_VERSION="1.55.7"
ARG STREAMLINK_VERSION="7.4.0"

#
# API Build
#
FROM golang:1.24-bookworm AS build-api
ARG GIT_SHA
ARG GIT_TAG
ENV GIT_SHA=$GIT_SHA
ENV GIT_TAG=$GIT_TAG
RUN echo "GIT_SHA=$GIT_SHA"
RUN echo "GIT_TAG=$GIT_TAG"
RUN apt update && apt install -y make git
WORKDIR /app
COPY . .
RUN make build_server build_worker

#
# API Tools
#
FROM debian:bookworm-slim AS tools

ARG STREAMLINK_VERSION

WORKDIR /tmp
RUN apt-get update && apt-get install -y --no-install-recommends \
    unzip git ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

# Download TwitchDownloader for the correct platform
ARG TWITCHDOWNLOADER_VERSION
ENV TWITCHDOWNLOADER_URL=https://github.com/lay295/TwitchDownloader/releases/download/${TWITCHDOWNLOADER_VERSION}/TwitchDownloaderCLI-${TWITCHDOWNLOADER_VERSION}-Linux-x64.zip


RUN if [ "$(uname -m)" = "aarch64" ]; then \
    TWITCHDOWNLOADER_URL=https://github.com/lay295/TwitchDownloader/releases/download/${TWITCHDOWNLOADER_VERSION}/TwitchDownloaderCLI-${TWITCHDOWNLOADER_VERSION}-LinuxArm64.zip; \
    fi && \
    echo "Download URL: $TWITCHDOWNLOADER_URL" && \
    curl -L $TWITCHDOWNLOADER_URL -o twitchdownloader.zip && \
    unzip twitchdownloader.zip && \
    rm twitchdownloader.zip

RUN git clone --depth 1 https://github.com/xenova/chat-downloader.git

#
# Frontend base
#
FROM node:22-alpine AS base-frontend

# Install dependencies only when needed
FROM node:22-alpine AS deps

RUN apk add --no-cache libc6-compat
WORKDIR /app

COPY frontend/package.json frontend/package-lock.json* ./
RUN \
    if [ -f yarn.lock ]; then yarn --frozen-lockfile; \
    elif [ -f package-lock.json ]; then npm ci --force; \
    elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm i --frozen-lockfile; \
    else echo "Lockfile not found." && exit 1; \
    fi

#
# Frontend build
#
FROM node:22-alpine AS build-frontend

WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY frontend/. .

ENV NEXT_TELEMETRY_DISABLED=1

RUN \
    if [ -f yarn.lock ]; then yarn run build; \
    elif [ -f package-lock.json ]; then npm run build; \
    elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm run build; \
    else echo "Lockfile not found." && exit 1; \
    fi

#
# Tests stage. Inclues depedencies required for tests
#
FROM golang:1.24-bookworm AS tests

ARG STREAMLINK_VERSION

RUN apt-get update && apt-get install -y --no-install-recommends python3 python3-pip ffmpeg make git

RUN pip3 install --upgrade pip streamlink==${STREAMLINK_VERSION} --break-system-packages

# Copy and install chat-downloader
COPY --from=tools /tmp/chat-downloader /tmp/chat-downloader
RUN cd /tmp/chat-downloader && python3 setup.py install && cd .. && rm -rf chat-downloader

# Setup fonts
RUN chmod 644 /usr/share/fonts/* && chmod -R a+rX /usr/share/fonts

# Copy TwitchDownloaderCLI
COPY --from=tools /tmp/TwitchDownloaderCLI /usr/local/bin/
RUN chmod +x /usr/local/bin/TwitchDownloaderCLI

# Production stage
FROM debian:bookworm-slim

ARG STREAMLINK_VERSION

WORKDIR /opt/app

# Install dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip fontconfig ffmpeg tzdata procps supervisor \
    fonts-noto-core fonts-noto-cjk fonts-noto-extra fonts-inter \
    curl \
    && rm -rf /var/lib/apt/lists/* \
    && ln -sf python3 /usr/bin/python

# Install pip packages
RUN pip3 install --no-cache-dir --upgrade pip streamlink==${STREAMLINK_VERSION} --break-system-packages

# Install gosu
RUN curl -LO https://github.com/tianon/gosu/releases/latest/download/gosu-$(dpkg --print-architecture | awk -F- '{ print $NF }') \
    && chmod 0755 gosu-$(dpkg --print-architecture | awk -F- '{ print $NF }') \
    && mv gosu-$(dpkg --print-architecture | awk -F- '{ print $NF }') /usr/local/bin/gosu

# Install node for frontend
ENV NODE_VERSION=22.x \
    DEBIAN_FRONTEND=noninteractive

# Install required packages, add NodeSource repository, and install Node.js
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    gnupg \
    && curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION} | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
RUN node --version && npm --version

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

# Copy api and worker builds
COPY --from=build-api /app/ganymede-api .
COPY --from=build-api /app/ganymede-worker .

# Setup frontend
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

COPY --from=build-frontend /app/public ./public

RUN mkdir .next
RUN chown nextjs:nodejs .next

COPY --from=build-frontend --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=build-frontend --chown=nextjs:nodejs /app/.next/static ./.next/static
ENV HOSTNAME="0.0.0.0" 


# Setup entrypoint
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh
COPY supervisord.conf /opt/app/supervisord.conf

EXPOSE 4000
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
