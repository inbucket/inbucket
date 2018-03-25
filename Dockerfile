# Docker build file for Inbucket, see https://www.docker.io/
# Inbucket website: http://www.inbucket.org/

FROM golang:1.10-alpine

# Configuration
ENV INBUCKET_SRC $GOPATH/src/github.com/jhillyerd/inbucket
ENV INBUCKET_HOME /opt/inbucket
ENV INBUCKET_SMTP_DOMAINNOSTORE bitbucket.local
ENV INBUCKET_SMTP_TIMEOUT 30s
ENV INBUCKET_POP3_TIMEOUT 30s
ENV INBUCKET_WEB_UIDIR $INBUCKET_HOME/ui
ENV INBUCKET_WEB_GREETINGFILE /config/greeting.html
ENV INBUCKET_WEB_COOKIEAUTHKEY secret-inbucket-session-cookie-key
ENV INBUCKET_STORAGE_TYPE file
ENV INBUCKET_STORAGE_PARAMS path:/storage
ENV INBUCKET_STORAGE_RETENTIONPERIOD 72h
ENV INBUCKET_STORAGE_MAILBOXMSGCAP 300

# Ports: SMTP, HTTP, POP3
EXPOSE 2500 9000 1100

# Persistent Volumes
VOLUME /config
VOLUME /storage

WORKDIR $INBUCKET_HOME
ENTRYPOINT "/start-inbucket.sh"

# Build Inbucket
COPY . $INBUCKET_SRC/
RUN "$INBUCKET_SRC/etc/docker/install.sh"
