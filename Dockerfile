FROM alpine:latest as builder

ARG GOARCH=adm64

ARG VERSION=dev

ARG SKIP_INIT_WEB

ENV SKIP_INIT_WEB=${SKIP_INIT_WEB}

ENV BUILD_CONFIG=script/build.config.sh

WORKDIR /synctv

COPY ./ ./

RUN apk add --no-cache bash curl git go g++

RUN bash script/build.sh --version=${VERSION} \
    --disable-micro --bin-name-no-suffix \
    --force-gcc='gcc -static' --force-g++='g++ -static' \
    --more-go-cmd-args='-a -v'

FROM alpine:latest

ENV PUID=0 PGID=0 UMASK=022

COPY --from=builder /synctv/build/synctv /usr/local/bin/synctv

RUN apk add --no-cache bash ca-certificates su-exec tzdata && \
    rm -rf /var/cache/apk/*

COPY script/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh && \
    mkdir -p /root/.synctv

WORKDIR /root/.synctv

EXPOSE 8080/tcp

VOLUME [ "/root/.synctv" ]

ENTRYPOINT [ "/entrypoint.sh" ]

CMD [ "server" ]
