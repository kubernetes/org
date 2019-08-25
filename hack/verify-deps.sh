#!/bin/bash
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

set -o nounset
set -o errexit
set -o pipefail

prefix="k8s.io/org"
repo_root="$(git rev-parse --show-toplevel)"
cd "${repo_root}"

_tmpdir="$(mktemp -d -t verify-deps.XXXXXX)"
cd "${_tmpdir}"
_tmpdir="$(realpath pwd)"

trap "rm -rf ${_tmpdir}" EXIT

_tmp_gopath="${_tmpdir}/go"
_tmp_repo_root="${_tmp_gopath}/src/${prefix}"
mkdir -p "${_tmp_repo_root}/.."
cp -a "${repo_root}" "${_tmp_repo_root}/.."

cd "${_tmp_repo_root}"
export PATH="${_tmp_gopath}/bin:${PATH}"
./hack/update-deps.sh

diff=$(diff -Nupr \
  -x ".git" \
  -x "bazel-*" \
  -x "_output" \
  "${repo_root}" "${_tmp_repo_root}" 2>/dev/null || true)

if [[ -n "${diff}" ]]; then
  echo "${diff}" >&2
  echo >&2
  echo "ERROR: bad deps. Run ./hack/update-deps.sh" >&2
  exit 1
fi
