FROM golang:1.20 AS build-stage-01

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -X main.Version=${VERSION} -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` -X main.GitHash=`git rev-parse HEAD`" -o ganymede-api cmd/server/main.go

FROM alpine:latest AS build-stage-02

RUN apk add --update --no-cache unzip git

WORKDIR /tmp
RUN wget https://github.com/rsms/inter/releases/download/v3.19/Inter-3.19.zip && unzip Inter-3.19.zip
RUN wget https://github.com/lay295/TwitchDownloader/releases/download/1.52.8/TwitchDownloaderCLI-1.52.8-LinuxAlpine-x64.zip && unzip TwitchDownloaderCLI-1.52.8-LinuxAlpine-x64.zip

RUN git clone https://github.com/xenova/chat-downloader.git

FROM alpine:latest AS production

# install packages
RUN apk add --update --no-cache python3 fontconfig icu-libs python3-dev gcc g++ ffmpeg bash tzdata shadow su-exec && ln -sf python3 /usr/bin/python
RUN python3 -m ensurepip
RUN pip3 install --no-cache --upgrade pip streamlink

# temp fix for Streamlink https://github.com/streamlink/streamlink/issues/5324
RUN pip3 install urllib3==1.26.15

# setup user
RUN groupmod -g 1000 users && \
  useradd -u 911 -U -d /data abc && \
  usermod -G users abc

# Install chat-downloader
COPY --from=build-stage-02 /tmp/chat-downloader /tmp/chat-downloader
RUN cd /tmp/chat-downloader && python3 setup.py install && cd .. && rm -rf chat-downloader

# install font
ENV INTER_PATH "/tmp/Inter Desktop/Inter-Regular.otf"
COPY --from=build-stage-02 ${INTER_PATH} /tmp/
RUN mkdir /usr/share/fonts/ && chmod a+rX /usr/share/fonts/
RUN mv /tmp/Inter-Regular.otf /usr/share/fonts/Inter.otf && fc-cache -f -v

# Install fallback fonts for chat rendering
RUN apk add terminus-font ttf-inconsolata ttf-dejavu font-noto font-noto-cjk ttf-font-awesome font-noto-extra font-noto-arabic

RUN chmod 644 /usr/share/fonts/* && chmod -R a+rX /usr/share/fonts

# TwitchDownloaderCLI
COPY --from=build-stage-02 /tmp/TwitchDownloaderCLI /usr/local/bin/
RUN chmod +x /usr/local/bin/TwitchDownloaderCLI

WORKDIR /opt/app

COPY --from=build-stage-01 /app/ganymede-api .

EXPOSE 4000

# copy entrypoint
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
