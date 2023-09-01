#!/usr/bin/env bash

function upload_remote_command() {
    filepath=$1
    contents=$(cat /var/lib/kurl/remotes/onenode/$filepath)
    curl -X PUT -f -H "Content-Type: application/json" -d "{\"command\": \"$contents\"}" "$TESTGRID_APIENDPOINT/v1/instance/$NODE_ID/upgrade-command/$filepath"
}

function upload_remote_commands() {
    export -f upload_remote_command
    while true
    do
           touch  ./lastwatch
           sleep 10
           find /var/lib/kurl/remotes/onenode -cnewer ./lastwatch -exec bash -c "upload_remote_command \$1" {} \;
    done
}

function main()
{
    upload_remote_commands
}

main
