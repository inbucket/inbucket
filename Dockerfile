# Docker build file for Inbucket, see https://www.docker.io/
# Inbucket website: http://inbucket.org/
FROM crosbymichael/golang
MAINTAINER James Hillyerd, @jameshillyerd

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

# Start Inbucket (WORKDIR doesn't support env vars)
WORKDIR /opt/inbucket
ENTRYPOINT ["./inbucket"]
CMD ["inbucket.conf"]
