# Docker build file for Inbucket, see https://www.docker.io/
# Inbucket website: http://www.inbucket.org/

FROM golang:1.5
MAINTAINER James Hillyerd, @jameshillyerd

# Configuration (WORKDIR doesn't support env vars)
ENV INBUCKET_SRC $GOPATH/src/github.com/jhillyerd/inbucket
ENV INBUCKET_HOME /opt/inbucket
WORKDIR /opt/inbucket
ENTRYPOINT ["bin/inbucket"]
CMD ["/etc/opt/inbucket.conf"]

# Ports: SMTP, HTTP, POP3
EXPOSE 10025 10080 10110

# Build Inbucket
ADD . $INBUCKET_SRC/
RUN "$INBUCKET_SRC/etc/docker/install.sh"
