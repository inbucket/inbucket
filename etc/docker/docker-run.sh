#!/bin/sh
# docker-run.sh
# description: Launch Inbucket's docker image

docker run -p 9000:10080 -p 2500:10025 -p 1100:10110 jhillyerd/inbucket
