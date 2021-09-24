#!/usr/bin/env bash
# Copyright 2021 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
readonly REPO_ROOT

cd "${REPO_ROOT}"
diff=$(find . -name "*.go" \( -not -path '*/vendor/*' -prune \) -exec gofmt -s -d '{}' +)
if [[ -z "$diff" ]]; then
  exit 0
fi

echo "$diff"
echo
echo "ERROR: found unformatted go files, fix with:" >&2
echo "" 
echo "  ${REPO_ROOT}/hack/update-gofmt.sh" >&2
exit 1
