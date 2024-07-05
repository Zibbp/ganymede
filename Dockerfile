FROM golang:1.22-bookworm AS build-stage-01

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -X main.Version=${VERSION} -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` -X main.GitHash=`git rev-parse HEAD`" -o ganymede-api cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -X main.Version=${VERSION} -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` -X main.GitHash=`git rev-parse HEAD`" -o ganymede-worker cmd/worker/main.go

FROM debian:bookworm-slim AS build-stage-02

RUN apt update && apt install -y git wget unzip

WORKDIR /tmp
RUN wget https://github.com/lay295/TwitchDownloader/releases/download/1.54.7/TwitchDownloaderCLI-1.54.7-Linux-x64.zip && unzip TwitchDownloaderCLI-1.54.7-Linux-x64.zip 

RUN git clone https://github.com/xenova/chat-downloader.git

FROM debian:bookworm-slim AS production

# install packages
RUN apt update && apt install -y python3 python3-pip fontconfig ffmpeg tzdata curl procps
RUN ln -sf python3 /usr/bin/python

# RUN apk add --update --no-cache python3 fontconfig icu-libs python3-dev gcc g++ ffmpeg bash tzdata shadow su-exec py3-pip && ln -sf python3 /usr/bin/python
RUN pip3 install --no-cache --upgrade pip streamlink --break-system-packages

## Installing su-exec in debain/ubuntu container.
RUN  set -ex; \
     \
     curl -o /usr/local/bin/su-exec.c https://raw.githubusercontent.com/ncopa/su-exec/master/su-exec.c; \
     \
     gcc -Wall \
         /usr/local/bin/su-exec.c -o/usr/local/bin/su-exec; \
     chown root:root /usr/local/bin/su-exec; \
     chmod 0755 /usr/local/bin/su-exec; \
     rm /usr/local/bin/su-exec.c; \
     \
## Remove the su-exec dependency. It is no longer needed after building.
     apt-get purge -y --auto-remove curl libc-dev

# setup user
RUN useradd -u 911 -d /data abc && \
    usermod -a -G users abc

# Install chat-downloader
COPY --from=build-stage-02 /tmp/chat-downloader /tmp/chat-downloader
RUN cd /tmp/chat-downloader && python3 setup.py install && cd .. && rm -rf chat-downloader

# Install fallback fonts for chat rendering
RUN apt install -y  fonts-noto-core fonts-noto-cjk fonts-noto-extra fonts-inter

RUN chmod 644 /usr/share/fonts/* && chmod -R a+rX /usr/share/fonts

# TwitchDownloaderCLI
COPY --from=build-stage-02 /tmp/TwitchDownloaderCLI /usr/local/bin/
RUN chmod +x /usr/local/bin/TwitchDownloaderCLI

WORKDIR /opt/app

COPY --from=build-stage-01 /app/ganymede-api .
COPY --from=build-stage-01 /app/ganymede-worker .

EXPOSE 4000

# copy entrypoint
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
