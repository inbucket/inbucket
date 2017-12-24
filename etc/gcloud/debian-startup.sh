#!/bin/bash
# setup.sh
# description: Install Inbucket on Google Cloud debian9 instance

inbucket_rel="1.2.0-rc2"
inbucket_pkg="inbucket_${inbucket_rel}_linux_amd64"
inbucket_url="https://dl.bintray.com/content/jhillyerd/golang/${inbucket_pkg}.tar.gz?direct"

fauxmailer_rel="0.1"
fauxmailer_pkg="fauxmailer_${fauxmailer_rel}_linux_amd64"
fauxmailer_url="https://github.com/jhillyerd/fauxmailer/releases/download/0.1/${fauxmailer_pkg}.tar.gz"

set -eo pipefail
[ $TRACE ] && set -x

# Prerequisites
apt-get --yes update
apt-get --yes install curl libcap2-bin
id inbucket &>/dev/null || useradd -r -m inbucket

# Extract
cd /opt
curl --location "$inbucket_url" | tar xzvf - 
ln -s "$inbucket_pkg/" inbucket

# Install
cd /opt/inbucket/etc/ubuntu
install -o inbucket -g inbucket -m 775 -d /var/opt/inbucket
touch /var/log/inbucket.log
chown inbucket: /var/log/inbucket.log
install -o root -g root -m 644 inbucket.logrotate /etc/logrotate.d/inbucket
install -o root -g root -m 644 inbucket.service /lib/systemd/system/inbucket.service
install -o root -g root -m 644 ../unix-sample.conf /etc/opt/inbucket.conf

# Setup
setcap 'cap_net_bind_service=+ep' /opt/inbucket/inbucket
curl -sL -o /opt/inbucket/themes/greeting.html "http://metadata.google.internal/computeMetadata/v1/instance/attributes/greeting" -H "Metadata-Flavor: Google"
systemctl enable inbucket.service
systemctl start inbucket

# Fauxmailer
cd /opt
curl --location "$fauxmailer_url" | tar xzvf - 
ln -s "$fauxmailer_pkg/" fauxmailer
