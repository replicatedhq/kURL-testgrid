#!/usr/bin/env bash

function upload_remote_command() {
    filepath=$1

    # if filepath is a directory, do not send it
    if [ -d "$filepath" ]; then
        return
    fi

    contents=$(cat $filepath | base64)

    # get last part of the filepath
    local filename=
    filename=$(basename -- "$filepath")

    curl -X PUT -f -H "Content-Type: application/json" -d "{\"command\": \"$contents\"}" "$TESTGRID_APIENDPOINT/v1/instance/$TEST_ID/upgrade-command/$filename"
}

function upload_remote_commands() {
    export -f upload_remote_command
    while true
    do
           touch  ./lastwatch
           sleep 10
           find /var/lib/kurl/remotes/onenode -cnewer ./lastwatch -exec bash -c "upload_remote_command \$0" {} \;
    done
}

function main()
{
    source /opt/kurl-testgrid/vars.sh
    upload_remote_commands
}

main
