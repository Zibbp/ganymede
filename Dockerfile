FROM golang:1.18 AS build-stage-01

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux go build -o ganymede-api cmd/server/main.go

FROM alpine:latest AS build-stage-02

RUN apk add --update --no-cache unzip

WORKDIR /tmp
RUN wget https://github.com/rsms/inter/releases/download/v3.19/Inter-3.19.zip && unzip Inter-3.19.zip
RUN wget https://github.com/lay295/TwitchDownloader/releases/download/1.50.6/TwitchDownloaderCLI-LinuxAlpine-x64.zip && unzip TwitchDownloaderCLI-LinuxAlpine-x64.zip

FROM alpine:latest AS production

ENV CHAT_DOWNLOADER_VER=0.2.1

RUN apk add --update --no-cache python3 fontconfig icu-libs python3-dev gcc g++ ffmpeg bash && ln -sf python3 /usr/bin/python
RUN python3 -m ensurepip
RUN pip3 install --no-cache --upgrade pip streamlink

# Install forked version of chat-downloader
RUN wget https://github.com/Zibbp/chat-downloader/archive/refs/tags/v${CHAT_DOWNLOADER_VER}.tar.gz
RUN tar -xvf v${CHAT_DOWNLOADER_VER}.tar.gz && cd chat-downloader-${CHAT_DOWNLOADER_VER} && python3 setup.py install && cd .. && rm -f v${CHAT_DOWNLOADER_VER}.tar.gz && rm -rf chat-downloader-${CHAT_DOWNLOADER_VER}

# Inter font install
ENV INTER_PATH "/tmp/Inter Desktop/Inter-Regular.otf"
COPY --from=build-stage-02 ${INTER_PATH} /tmp/
RUN mkdir -p /usr/share/fonts/opentype/ && install -m644 /tmp/Inter-Regular.otf /usr/share/fonts/opentype/Inter.otf && rm ./tmp/Inter-Regular.otf && fc-cache -fv

# TwitchDownloaderCLI
COPY --from=build-stage-02 /tmp/TwitchDownloaderCLI /usr/local/bin/
RUN chmod +x /usr/local/bin/TwitchDownloaderCLI

COPY --from=build-stage-01 /app/ganymede-api .

CMD ["./ganymede-api"]
