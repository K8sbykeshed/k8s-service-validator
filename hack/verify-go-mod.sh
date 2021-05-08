
set -o errexit
set -o nounset
set -o pipefail

go mod tidy
git diff --exit-code
