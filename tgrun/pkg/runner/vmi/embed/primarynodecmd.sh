#!/usr/bin/env bash

source /opt/kurl-testgrid/common.sh

function runJoinCommand()
{
  joinCommand=$(get_join_command)
  primaryJoin=$(echo "$joinCommand" | sed 's/{.*primaryJoin":"*\([0-9a-zA-Z=]*\)"*,*.*}/\1/' | base64 -d)
  primaryJoin+=" yes" # assume yes for prompts
  eval $primaryJoin
  KURL_EXIT_STATUS=$?
}

function runAirgapJoinCommand()
{
  retry 5 download_and_verify_tarball "$KURL_URL" install.tar.gz
  tar -xzf install.tar.gz
  joinCommand=$(get_join_command)
  primaryJoin=$(echo "$joinCommand" | sed 's/{.*primaryJoin":"*\([0-9a-zA-Z=]*\)"*,*.*}/\1/' | base64 -d)
  eval $primaryJoin
  KURL_EXIT_STATUS=$?
}

function main()
{
  green "setup runner"
  setup_runner

  green "report node in waiting for join command"
  report_status_update "waitJoinCommand"

  green "wait for join command"
  wait_for_join_commandready

  green "run join command"
  if [ "$(is_airgap)" = "1" ]; then
    runAirgapJoinCommand
  else
    runJoinCommand
  fi

  if [ $KURL_EXIT_STATUS -ne 0 ]; then
    report_status_update "failed"
    send_logs
    exit 1
  fi

  green "report success join"
  report_status_update "joined"
  send_logs

  # must stick around as part of the cluster until the test is complete
  green "wait for initprimary done"
  wait_for_initprimary_done

  report_status_update "success"
  send_logs
}

main
