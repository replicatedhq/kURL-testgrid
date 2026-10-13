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
fi
