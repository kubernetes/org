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
  MadhavJivrajani
  mrbobbytables
  nikhita
  palnabarun
  Priyankasaggu11929
)

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
  "${admins[@]/#/--required-admins=}"
)

"${cmd}" "${args[@]}" "${@}"
