#!/bin/bash
# M10-BASE-A3: Nightly md-doctor + slo-report cron job.
# Schedule: 0 3 * * * /opt/ant/deploy/cron/nightly.sh
set -euo pipefail
cd /opt/ant
make ci-nightly 2>&1 | tee /var/log/ant/ci-nightly-$(date +%Y%m%d).log
