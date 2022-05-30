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

make config
function urlencode() {
          sed ':begin;$!N;s/\n/%0A/;tbegin'
}

if [[ -z "$(git status --porcelain)" ]]; then
    echo "kubernetes/org is up to date."
else
    # Print it both ways because sometimes we haven't modified the file (e.g. zz_generated),
    # and sometimes we have (e.g. configmap checksum).
    echo "Found diffs in: $(git diff-index --name-only HEAD --)"
    for x in $(git diff-index --name-only HEAD --); do
        echo "file=$x Please run make config to ensure OWNER_ALIASES are up to date. $(git diff $x | urlencode)"
    done
    echo "kubernetes/org is out of date. Please run make config to ensure OWNER_ALIASES are up to date."
    exit 1
fi
