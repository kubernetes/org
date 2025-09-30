#!/usr/bin/env bash
# Copyright 2018 The Kubernetes Authors.
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
set -x

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
readonly REPO_ROOT

readonly admins=(
  cblecker
  jasonbraganza
  MadhavJivrajani
  mrbobbytables
  nikhita
  palnabarun
  Priyankasaggu11929
)

# this is the hourly token limit for the GitHub API
# if unset, the default is set in the peribolos code: https://github.com/kubernetes-sigs/prow/blob/0bca2f1416a9c15d75b9cee8704b56b38d5895c6/prow/cmd/peribolos/main.go#L41
# if set to 0, rate limiting is disabled
readonly HOURLY_TOKENS=3000

cd "${REPO_ROOT}"
make update-prep
cmd="${REPO_ROOT}/_output/bin/peribolos"
args=(
  --config-path="${REPO_ROOT}/_output/gen-config.yaml"
  --fix-org
  --fix-org-members
  --fix-teams
  --fix-team-members
  --fix-team-repos
  --github-hourly-tokens="${HOURLY_TOKENS}"
  "${admins[@]/#/--required-admins=}"
)

"${cmd}" "${args[@]}" "${@}"
