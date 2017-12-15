# Docker build file for Inbucket, see https://www.docker.io/
# Inbucket website: http://www.inbucket.org/

FROM golang:1.9-alpine
MAINTAINER James Hillyerd, @jameshillyerd

# Configuration (WORKDIR doesn't support env vars)
ENV INBUCKET_SRC $GOPATH/src/github.com/jhillyerd/inbucket
ENV INBUCKET_HOME /opt/inbucket
WORKDIR $INBUCKET_HOME
ENTRYPOINT ["/con/context/start-inbucket.sh"]
CMD ["/con/configuration/inbucket.conf"]

# Ports: SMTP, HTTP, POP3
EXPOSE 10025 10080 10110

# Persistent Volumes, following convention at:
#   https://github.com/docker/docker/issues/9277
# NOTE /con/context is also used, not exposed by default
VOLUME /con/configuration
VOLUME /con/data

# Build Inbucket
COPY . $INBUCKET_SRC/
RUN "$INBUCKET_SRC/etc/docker/install.sh"
