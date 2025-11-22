# Docker build file for Inbucket: https://www.inbucket.org/

### Build frontend
# Due to no official elm compiler for arm; build frontend with amd64.
FROM --platform=linux/amd64 node:20 AS frontend
RUN npm install -g node-gyp
WORKDIR /build
COPY . .
WORKDIR /build/ui
RUN rm -rf .parcel-cache dist elm-stuff node_modules
RUN yarn install --frozen-lockfile --non-interactive
RUN yarn run build

### Build backend
FROM golang:1.25-alpine3.22 AS backend
RUN apk add --no-cache --virtual .build-deps g++ git make
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0
RUN make clean deps
RUN go build -o inbucket \
  -ldflags "-X 'main.version=$(git describe --tags --always)' -X 'main.date=$(date -Iseconds)'" \
  -v ./cmd/inbucket

### Run in minimal image
FROM alpine:3.22
RUN apk --no-cache add tzdata
WORKDIR /opt/inbucket
RUN mkdir bin defaults ui
COPY --from=backend /build/inbucket bin
COPY --from=frontend /build/ui/dist ui
COPY etc/docker/defaults/greeting.html defaults
COPY etc/docker/defaults/start-inbucket.sh /

# Configuration
ENV INBUCKET_SMTP_DISCARDDOMAINS=bitbucket.local
ENV INBUCKET_SMTP_TIMEOUT=30s
ENV INBUCKET_POP3_TIMEOUT=30s
ENV INBUCKET_WEB_GREETINGFILE=/config/greeting.html
ENV INBUCKET_WEB_COOKIEAUTHKEY=secret-inbucket-session-cookie-key
ENV INBUCKET_WEB_UIDIR=ui
ENV INBUCKET_STORAGE_TYPE=file
ENV INBUCKET_STORAGE_PARAMS=path:/storage
ENV INBUCKET_STORAGE_RETENTIONPERIOD=72h
ENV INBUCKET_STORAGE_MAILBOXMSGCAP=300

# Healthcheck
HEALTHCHECK --interval=5s --timeout=5s --retries=3 CMD /bin/sh -c 'wget localhost:$(echo ${INBUCKET_WEB_ADDR:-0.0.0.0:9000}|cut -d: -f2) -q -O - >/dev/null'

# Ports: SMTP, HTTP, POP3
EXPOSE 2500 9000 1100

# Persistent Volumes
VOLUME /config
VOLUME /storage

ENTRYPOINT ["/start-inbucket.sh"]
CMD ["-logjson"]
