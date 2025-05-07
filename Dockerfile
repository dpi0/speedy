FROM golang:1.24.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -o speedy .

FROM alpine:3

ARG VERSION=1.2.0

RUN apk add --no-cache --virtual .deps tar curl && \
    ARCH=$(apk info --print-arch) && \
    case "$ARCH" in \
        x86)    _arch=i386      ;; \
        armv7)  _arch=armhf     ;; \
        *)      _arch="$ARCH"   ;; \
    esac && \
    cd /tmp && \
    curl -sSL https://install.speedtest.net/app/cli/ookla-speedtest-${VERSION}-linux-${_arch}.tgz | tar xz && \
    mv /tmp/speedtest /usr/local/bin/ && \
    rm -rf /tmp/speedtest.* && \
    apk del .deps

COPY --from=builder /app/speedy /usr/local/bin/

RUN adduser -D -u 1000 speedy
USER speedy

ENV SPEEDTEST_CRON="*/2 * * * *"

EXPOSE 8080

CMD ["/usr/local/bin/speedy"]
