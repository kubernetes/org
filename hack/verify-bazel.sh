#!/usr/bin/env bash
# Copyright 2016 The Kubernetes Authors.
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

cd "$(git rev-parse --show-toplevel)"
ls ./vendor/BUILD.bazel >/dev/null
gazelle_diff=$(bazel run //:gazelle-diff)
kazel_diff=$(bazel run //:kazel-diff)

if [[ -n "${gazelle_diff}" ]]; then
  echo "Gazelle diff:"
  echo "${gazelle_diff}"
fi

if [[ -n "${kazel_diff}" ]]; then
  echo "Kazel diff:"
  echo "${kazel_diff}"
fi

if [[ -n "${gazelle_diff}${kazel_diff}" ]]; then
  echo "FAIL: invalid BUILD.bazel files. Fix with ./hack/update-bazel.sh"
  exit 1
fi
