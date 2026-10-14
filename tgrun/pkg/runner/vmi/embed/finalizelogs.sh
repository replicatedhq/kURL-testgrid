#!/usr/bin/env bash

function send_logs()
{
  cat /var/log/cloud-init-output.log | grep -v '"__CURSOR" :' > /tmp/testgrid-node-logs # strip junk
  curl -X PUT -f --connect-timeout 10 --max-time 60 --data-binary "@/tmp/testgrid-node-logs" "$TESTGRID_APIENDPOINT/v1/instance/$NODE_ID/node-logs"
}

echo "END OF LOGS" >> /var/log/cloud-init-output.log

# Hard self-timeout: an unbounded log upload here is exactly what leaves a
# -sendlogs VM stuck in Running for hours, since cloud-init's runcmd has no
# command timeout of its own (sc-140012). Bound the whole upload attempt so
# this VM reaches the poweroff step on its own even if the upload hangs or
# never completes.
if ! timeout 90 bash -c "$(declare -f send_logs); send_logs"; then
  echo "send_logs timed out or failed after 90s" >> /var/log/cloud-init-output.log
  # The line above is written to the same file send_logs just failed to ship, so without a
  # follow-up attempt it would never reach the API. Make one more attempt so the failure note
  # itself gets uploaded instead of being stranded on disk. The outer bound must exceed
  # send_logs' own curl --max-time (60s), otherwise this timeout always fires first and kills
  # a slow-but-otherwise-successful upload before curl's own flags get a chance to succeed or
  # fail cleanly.
  timeout 65 bash -c "$(declare -f send_logs); send_logs" || true
fi
