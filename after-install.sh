#!/bin/bash

if [ ! -f /etc/mailway/conf.d/server-id.yml ]; then
    UUID=$(cat /proc/sys/kernel/random/uuid)
    echo "server_id: $UUID" > /etc/mailway/conf.d/server-id.yml
    echo "generated server id: $UUID"
fi

systemctl daemon-reload
