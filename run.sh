#!/bin/sh

set -x

# This is the entrypoint for the image and meant to wrap the
# logic of gathering/reporting results to the Sonobuoy worker.

results_dir="${RESULTS_DIR:-/tmp/results}"

# saveResults prepares the results for handoff to the Sonobuoy worker.
# See: https://github.com/vmware-tanzu/sonobuoy/blob/master/docs/plugins.md
saveResults() {
	# Signal to the worker that we are done and where to find the results.
	printf "${results_dir}"/out.json > "${results_dir}"/done
}

# Ensure that we tell the Sonobuoy worker we are done regardless of results.
trap saveResults EXIT

./svc-test | tee "${results_dir}"/out.json
