# https://developers.home-assistant.io/docs/add-ons/configuration#add-on-dockerfile
ARG BUILD_FROM

# --- Build stage -------------------------------------------------------------
FROM golang:1.24-alpine AS build-stage
ARG BUILD_ARCH
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Map Home Assistant build architectures to Go targets
RUN case "${BUILD_ARCH}" in \
        amd64)   GOARCH=amd64 ;; \
        i386)    GOARCH=386 ;; \
        aarch64) GOARCH=arm64 ;; \
        armv7)   GOARCH=arm GOARM=7 ;; \
        armhf)   GOARCH=arm GOARM=6 ;; \
        *) echo "Unsupported BUILD_ARCH: ${BUILD_ARCH}" && exit 1 ;; \
    esac \
    && CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} GOARM=${GOARM:-} \
       go build -trimpath -ldflags="-s -w" -o /app/app ./cmd

# --- Runtime stage -----------------------------------------------------------
FROM ${BUILD_FROM}
COPY rootfs /
RUN chmod a+x /etc/services.d/setecna/run /etc/services.d/setecna/finish
COPY --from=build-stage /app/app /usr/bin/app
