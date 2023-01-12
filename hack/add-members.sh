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

set -euo pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

go install "$SCRIPT_ROOT"/cmd/korg

DRY_RUN=${DRY_RUN:-false}

if [[ -z ${WHO:-} ]]; then
   echo "No users specified. Specify the users you would like to add with WHO=user1,user2"
   exit 1
fi

[ -z ${REPOS+x} ] && echo "No repos specified. Defaulting to kubernetes."
REPOS=${REPOS:-"kubernetes"}

cd "$SCRIPT_ROOT"
for username in ${WHO//,/ }
do
    echo "Adding $username to $REPOS"
   if [ "$DRY_RUN" = true ]; then
     echo "Running in dry run mode."
     korg add "$username" --org "$REPOS"
   else
     korg add "$username" --org "$REPOS" --confirm
   fi
done
