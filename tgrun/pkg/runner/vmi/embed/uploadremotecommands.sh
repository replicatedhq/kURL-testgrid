#!/usr/bin/env bash

function upload_remote_commands() {
    while true
    do
           touch  ./lastwatch
           sleep 10
           find /YOUR/WATCH/PATH -cnewer ./lastwatch -exec SOMECOMMAND {} \;
    done
}
