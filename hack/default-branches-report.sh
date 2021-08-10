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

readonly kubernetes_orgs=(
    kubernetes
    kubernetes-sigs
    kubernetes-client
    kubernetes-csi
)

readonly gh_api_cmd=(
    gh api
    --field=per_page=100
    --paginate
    --method=GET
)

function ensure_dependencies() {
  if ! command -f gh >/dev/null; then
      >&2 echo "gh not found. Please install: https://cli.github.com/manual/installation"
      exit 1
  fi
}

function main() {
  local orgs=("$@")
  ensure_dependencies
  for org in "${orgs[@]}"; do
      echo "* ${org}"
      "${gh_api_cmd[@]}" "/orgs/${org}/repos" \
          --field=sort=full_name \
          --template \
          '{{range .}}  * [{{if eq .default_branch "master"}} {{else}}X{{end}}] [{{.full_name}}]({{.html_url}}) {{"\n"}}{{end}}'
  done
  echo
  echo "Manual inspection required if you want to link issues that tracked default branch name migration"
}

args=("${@:1}")
if [ ${#args[@]} -eq 0 ]; then 
  args=("${kubernetes_orgs[@]}")
fi

main "${args[@]}"
