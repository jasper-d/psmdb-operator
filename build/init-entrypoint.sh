#!/bin/bash

set -o errexit
set -o xtrace

if [ ! -z "$1" ]; then
  EXTERNAL_NAME="$HOSTNAME.$1"
  echo "Verify dns resolution for $EXTERNAL_NAME"

  DNS_IP=""
  COUNTER=0
  while [ -z "$DNS_IP" ] && [ $COUNTER -lt 30 ]; do
    DNS_IP=$(nslookup $EXTERNAL_NAME | awk '/^Address: / { print $2 }')
    [ -z "$DNS_IP" ] && sleep 10
    COUNTER=$((COUNTER+1))
  done
  echo "Counter: $COUNTER"
  echo "DNS_IP: $DNS_IP"
  if [ -z "$DNS_IP" ]; then
    echo "timeout resolving $EXTERNAL_NAME"
    exit 7
  else
    echo "Resolved $EXTERNAL_NAME to $DNS_IP"
  fi
fi

install -o "$(id -u)" -g "$(id -g)" -m 0755 -D /ps-entry.sh /data/db/ps-entry.sh
