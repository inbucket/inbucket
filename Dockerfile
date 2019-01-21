# Docker build file for Inbucket: https://www.inbucket.org/

# Build
FROM golang:1.11-alpine3.8 as builder
RUN apk add --no-cache --virtual .build-deps git make npm
WORKDIR /build
COPY . .
ENV CGO_ENABLED 0
RUN make clean deps
RUN go build -o inbucket \
  -ldflags "-X 'main.version=$(git describe --tags --always)' -X 'main.date=$(date -Iseconds)'" \
  -v ./cmd/inbucket
WORKDIR /build/ui
RUN rm -rf dist elm-stuff node_modules
RUN npm i
RUN npm run build

# Run in minimal image
FROM alpine:3.8
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

# Ports: SMTP, HTTP, POP3
EXPOSE 2500 9000 1100

# Persistent Volumes
VOLUME /config
VOLUME /storage

ENTRYPOINT ["/start-inbucket.sh"]
CMD ["-logjson"]
