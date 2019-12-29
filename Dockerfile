FROM golang:1.13-alpine3.10 as builder

RUN set -eux && \
	apk add --no-cache \
		git

WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -v -tags 'osusergo,netgo,static,static_build' -installsuffix netgo -ldflags '-d -s -w "-extldflags=-fno-PIC -static"' -o bb8bot .



FROM alpine:3.10

RUN set -eux && \
	apk add --no-cache \
		ca-certificates \
        bash

COPY --from=builder /build/bb8bot /usr/bin/bb8bot
COPY --from=builder /build/config.toml /etc/bb8bot/config.toml

ENTRYPOINT ["/usr/bin/bb8bot", "-c", "/etc/bb8bot/config.toml"]