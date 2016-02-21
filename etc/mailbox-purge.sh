#!/bin/sh
curl -i -H "Accept: application/json" --noproxy localhost -X DELETE http://localhost:9000/mailbox/$1
