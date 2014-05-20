# Docker build file for Inbucket, see https://www.docker.io/
# Inbucket website: http://inbucket.org/
FROM ubuntu:14.04
MAINTAINER James Hillyerd, @jameshillyerd

# To force the upgrade packages change REFRESHED_AT date, otherwise Docker
# will cache the old updates
ENV REFRESHED_AT 2014-05-19

# Update Ubuntu
RUN apt-get -q update \
  && DEBIAN_FRONTEND=noninteractive apt-get -qy upgrade

# Clean up APT when done.
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Install Inbucket
ENV INBUCKET_HOME /opt/inbucket
ADD inbucket $INBUCKET_HOME/inbucket
ADD themes $INBUCKET_HOME/themes
ADD etc/unix-sample.conf $INBUCKET_HOME/inbucket.conf

# Volume for mail data
VOLUME /var/opt/inbucket

# SMTP, HTTP, POP3 ports
EXPOSE 25
EXPOSE 80
EXPOSE 110

# Start Inbucket
CMD $INBUCKET_HOME/inbucket $INBUCKET_HOME/inbucket.conf
