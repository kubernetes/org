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


# This script removes members from all Kubernetes orgs by removing any
# occurrence of their GitHub handle from files contained within the /config
# directory. To preserve a clean git history, each removal is done as its own
# commit, mirroring the process done when a member is added.
#
# The script expects a file containing a list of GitHub handles, one per line.
#
# The environment variable `DRYRUN` controls whether the changes are simulated
# or live. The default, `false` will print the user, files modified, lines
# changed, and git commit command.
#
# ENV:
#   DRYRUN: {true,false} - default: false
# ARGS:
#   $1: path to a file containing a list of members to be removed
# USAGE:
#   DRYRUN={true,false} ./remove-members.sh <file>
# EXAMPLES:
#   DRYRUN=false ./remove-members.sh inactive-members.txt # Prints changes
#   DRYRUN=true ./remove-members.sh inactive-members.txt  # Removes members


set -o errexit
set -o nounset
set -o pipefail

readonly REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly CONFIG_PATH="$REPO_ROOT/config"
readonly DRYRUN="${DRYRUN:-true}"

if [ ! -f "$1" ]; then
  echo "No file specified."
  exit 1
fi

members=()
mapfile -t members < "$1"
for member in "${members[@]}"; do
  matches=()
  orgs=()
  teams=()

  # Assembles list of files containing member to be removed (\s+)?- ${member}(\s+|\s+?#.*)?$
  mapfile -t matches < <(grep -rliP  --include="*.yaml" "(\s+)?- $member(\s+|\s+?#.*)?$" "$CONFIG_PATH")

  for filename in "${matches[@]}"; do

    if [ "$(basename "$filename")" == "org.yaml" ]; then
      if [ "$DRYRUN" == "false" ]; then
        sed -E -i "/(\s+)?- $member(\s+|\s+?#.*)?$/Id" "$filename"
        git add "$filename"
       else
        grep -inHP "(\s+)?- $member(\s+|\s+?#.*)?$" "$filename"
      fi
      # Adds org component to array to build removal commit message
      orgs+=("$(basename "$(dirname "$filename")")")
    fi

    if [ "$(basename "$filename")" == "teams.yaml" ]; then
      if [ "$DRYRUN" == "false" ]; then
        sed -E -i "/(\s+)?- $member(\s+|\s+?#.*)?$/Id" "$filename"
        git add "$filename"
       else
         grep -inHP "(\s+)?- $member(\s+|\s+?#.*)?$" "$filename"
      fi
      # Adds team component to array to build removal commit message
      # It is not perfect as teams can be in org files, but it does make 
      # commit messages more descriptive when possible.
      teams+=("$(basename "$(dirname "$filename")")")
    fi
  done

  sorted_unique_orgs=()
  sorted_unique_teams=()

  # Removes duplicates and sorts to build a better commit message.
  mapfile -t sorted_unique_orgs < <(echo "${orgs[@]}" | tr ' ' '\n' | sort -u)
  mapfile -t sorted_unique_teams < <(echo "${teams[@]}" | tr ' ' '\n' | sort -u)



  org_commit_msg="Remove $member from the "
  if [[ "${#sorted_unique_orgs[@]}" -eq "1" ]]; then
    org_commit_msg+="${sorted_unique_orgs[*]} org"
  elif [[ "${#sorted_unique_orgs[@]}" -ge "1" ]]; then
    printf -v joined '%s, ' "${sorted_unique_orgs[@]}"
    org_commit_msg+="${joined%, } orgs"
  fi

  cmd="git commit -m \"$org_commit_msg\""
  if [[ "${sorted_unique_teams[0]}" != "" ]]; then
    for team in "${sorted_unique_teams[@]}"; do
      cmd+=" -m \"Remove $member from $team teams\""
    done
  fi

  if [ "$DRYRUN" == "false" ]; then
    eval "$cmd"
  else
    echo "Command: $cmd"
  fi
done
