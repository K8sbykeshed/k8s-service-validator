#!/bin/sh

set -x
results_dir="${SONOBUOY_RESULTS_DIR:-/tmp/sonobuoy/results}"
./svc-test --skip-labels "type=iperf" | tee "${results_dir}"/out.json
