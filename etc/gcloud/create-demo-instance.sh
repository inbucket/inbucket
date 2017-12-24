#!/bin/bash
# create-demo-instance.sh

set -x
gcloud compute instances create inbucket-1 --machine-type=f1-micro \
  --image-project=debian-cloud --image-family=debian-9 \
  --metadata-from-file=startup-script=debian-startup.sh,greeting=demo-greeting.html \
  --tags=http-server --address=inbucket-demo
