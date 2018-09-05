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

if [[ $# -lt 6 ]]; then
  echo "Usage: bazel run //admin:update -- --github-token-path ~/my/github/token # --confirm" >&2
  exit 1
fi

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

pushd "$(dirname "$(realpath "$BASH_SOURCE")")"
bazel test //config:all
popd
peribolos="$1"
shift
echo $@
"$peribolos" $@
