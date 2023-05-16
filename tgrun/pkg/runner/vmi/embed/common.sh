#!/usr/bin/env bash

set -x

function green()
{
    text="${1:-}"
    echo -e "\033[32m$text\033[0m"
}

function command_exists()
{
    command -v "$@" > /dev/null 2>&1
}

function ensure_ntp()
{
    if command_exists "apt-get" ; then
        apt-get update
        apt-get install -y systemd-timesyncd || true
        systemctl start systemd-timesyncd || true
    fi
    timedatectl set-ntp true
}

function setup_runner()
{
    setenforce 0 || true # rhel variants

    echo "vm.overcommit_memory = 1" > /etc/sysctl.d/99-custom.conf
    echo "kernel.panic = 10" >> /etc/sysctl.d/99-custom.conf
    echo "kernel.panic_on_oops = 1" >> /etc/sysctl.d/99-custom.conf
    sysctl -p /etc/sysctl.d/99-custom.conf

    echo "$TEST_ID" > /tmp/testgrid-id

    if [ ! -c /dev/urandom ]; then
        /bin/mknod -m 0666 /dev/urandom c 1 9 && /bin/chown root:root /dev/urandom
    fi

    echo "OS INFO:"
    cat /etc/*-release
    echo ""

    ensure_ntp
}

function send_logs()
{
  cat /var/log/cloud-init-output.log | grep -v '"__CURSOR" :' > /tmp/testgrid-node-logs # strip junk
  curl -X PUT -f --data-binary "@/tmp/testgrid-node-logs" "$TESTGRID_APIENDPOINT/v1/instance/$NODE_ID/node-logs"
}

function report_status_update()
{
  curl -X PUT -f -H "Content-Type: application/json" -d "{\"status\": \"$1\"}" "$TESTGRID_APIENDPOINT/v1/instance/$NODE_ID/node-status"
}

function get_initprimary_status()
{
  primaryNodeId="${TEST_ID}-initialprimary"
  response=$(curl -X GET -f "$TESTGRID_APIENDPOINT/v1/instance/$primaryNodeId/node-status")
  primaryNodeStatus=$(echo "$response" | sed 's/{.*status":"*\([0-9a-zA-Z]*\)"*,*.*}/\1/')
  echo "$primaryNodeStatus"
}

function get_join_command()
{
  joinCommand=$(curl -X GET -f "$TESTGRID_APIENDPOINT/v1/instance/$TEST_ID/join-command")
  echo "${joinCommand}"
}

function wait_for_join_commandready()
{
  i=0
  while true; do
    primaryNodeStatus=$(get_initprimary_status)
    if [[ "$primaryNodeStatus" = "joinCommandStored" ]] ; then
      echo "join command is ready"
      break
    elif [[ "$primaryNodeStatus" = "failed" ]] ; then
      echo "primaryNodeStatus failed"
      report_status_update "failed"
      send_logs
      exit 1
    fi
    echo "join command not ready"
    i=$((i+1))
    # it could take up to 30 minutes to run the initial primary install script
    # and an additional 10 minutes for OL to run centos2ol script
    if [ $i -gt 40 ]; then
      echo "wait_for_join_commandready timeout"
      report_status_update "failed"
      send_logs
      exit 1
    fi
    sleep 60
  done
}

function wait_for_initprimary_done()
{
  i=0
  while true; do
    primaryNodeStatus=$(get_initprimary_status)
    if [[ "$primaryNodeStatus" = "success" ]]; then
      echo "initprimary status finsihed the test"
      break
    elif [[ "$primaryNodeStatus" = "failed" ]] ; then
      echo "primaryNodeStatus failed"
      report_status_update "failed"
      send_logs
      exit 1
    fi
    echo "initprimary not ready"
    i=$((i+1))
    # we give the upgrade 180 minutes to finish plus 30 minutes padding for the cluster to become ready
    if [ $i -gt 210 ]; then
      echo "wait_for_initprimary_done timeout"
      report_status_update "failed"
      send_logs
      exit 1
    fi
    sleep 60
  done
}

function is_airgap()
{
  local airgap=0
  if echo "$KURL_URL" | grep -q "\.tar\.gz$" ; then
    airgap=1
  fi
  echo $airgap
}

function retry() {
    local retries=$1
    shift
    local count=0
    until "$@"; do
        exit_code=$?
        count=$((count + 1))
        if [ $count -lt "$retries" ]; then
            echo "Retry $count/$retries exited $exit_code, retrying in 15 seconds..."
            sleep 1
        else
            echo "Retry $count/$retries exited $exit_code, no more retries left."
            return $exit_code
        fi
    done
}

function download_and_verify_tarball() {
    local url=$1
    local outfile=$2
    echo "Downloading $url to $outfile"
    curl -fsSL -o "$outfile" "$url"
    exit_code=$?
    if [ $exit_code -ne 0 ]; then
        return $exit_code
    fi
    tar -tzf "$outfile" >/dev/null
}
