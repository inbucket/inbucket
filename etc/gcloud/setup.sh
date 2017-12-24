#!/bin/bash
# setup.sh
# description: Install Inbucket on Google Cloud debian9 instance

release="inbucket_1.2.0-rc2_linux_amd64"
url="https://dl.bintray.com/content/jhillyerd/golang/${release}.tar.gz?direct"

set -eo pipefail
[ $TRACE ] && set -x

# Prerequisites
apt-get --yes update
apt-get --yes install curl libcap2-bin
id inbucket &>/dev/null || useradd -r -m inbucket

# Extract
cd /opt
curl --location "$url" | tar xzvf - 
ln -s "$release/" inbucket

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
systemctl enable inbucket.service
systemctl start inbucket
