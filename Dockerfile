# Docker build file for Inbucket: https://www.inbucket.org/

# Install build-time dependencies
FROM golang:1.14-alpine3.11 as builder
RUN apk add --no-cache --virtual .build-deps g++ git make npm python3
WORKDIR /build
COPY . .
ENV CGO_ENABLED 0
RUN make clean deps
WORKDIR /build/ui
RUN rm -rf dist elm-stuff node_modules
RUN npm ci
ADD https://github.com/elm/compiler/releases/download/0.19.1/binary-for-linux-64-bit.gz elm.gz
RUN gunzip elm.gz && chmod 755 elm && mv elm /usr/bin/

# Build server
WORKDIR /build
RUN go build -o inbucket \
  -ldflags "-X 'main.version=$(git describe --tags --always)' -X 'main.date=$(date -Iseconds)'" \
  -v ./cmd/inbucket

# Build frontend
WORKDIR /build/ui
RUN npm run build

# Run in minimal image
FROM alpine:3.11
WORKDIR /opt/inbucket
RUN mkdir bin defaults ui
COPY --from=builder /build/inbucket bin
COPY --from=builder /build/ui/dist ui
COPY etc/docker/defaults/greeting.html defaults
COPY etc/docker/defaults/start-inbucket.sh /

# Configuration
ENV INBUCKET_SMTP_DISCARDDOMAINS bitbucket.local
ENV INBUCKET_SMTP_TIMEOUT 30s
ENV INBUCKET_POP3_TIMEOUT 30s
ENV INBUCKET_WEB_GREETINGFILE /config/greeting.html
ENV INBUCKET_WEB_COOKIEAUTHKEY secret-inbucket-session-cookie-key
ENV INBUCKET_WEB_UIDIR=ui
ENV INBUCKET_STORAGE_TYPE file
ENV INBUCKET_STORAGE_PARAMS path:/storage
ENV INBUCKET_STORAGE_RETENTIONPERIOD 72h
ENV INBUCKET_STORAGE_MAILBOXMSGCAP 300

# Healthcheck
HEALTHCHECK --interval=5s --timeout=5s --retries=3 CMD /bin/sh -c 'wget localhost:$(echo ${INBUCKET_WEB_ADDR:-0.0.0.0:9000}|cut -d: -f2) -q -O - >/dev/null'

# Ports: SMTP, HTTP, POP3
EXPOSE 2500 9000 1100

# Persistent Volumes
VOLUME /config
VOLUME /storage

ENTRYPOINT ["/start-inbucket.sh"]
CMD ["-logjson"]
