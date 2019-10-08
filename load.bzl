# Copyright 2019 The Kubernetes Authors.
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

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

def repositories():
    git_repository(
        name = "io_k8s_repo_infra",
        commit = "db6ceb5f992254db76af7c25db2edc5469b5ea82",
        remote = "https://github.com/kubernetes/repo-infra.git",
        shallow_since = "1570128715 -0700",
    )
