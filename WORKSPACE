# gazelle:repository_macro repos.bzl%go_repositories
workspace(name = "io_k8s_org")

load("//:load.bzl", "repositories")

repositories()

load("@io_k8s_repo_infra//:load.bzl", repo_infra_repos = "repositories")

repo_infra_repos()

load("@io_k8s_repo_infra//:repos.bzl", "configure")

configure(go_modules = None)

load("//:repos.bzl", "go_repositories")

go_repositories()

load("@io_k8s_repo_infra//:repos.bzl", _repo_infra_go_repos = "go_repositories")

_repo_infra_go_repos()

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_k8s_test_infra",
    sha256 = "bbb44b86e5ec7002bfd0e19bd5921db841f70f0e3bab738986b35f9e91d85928",
    strip_prefix = "test-infra-6304ab49bc654eb0799d57ae2b076a246cb2051f",
    urls = ["https://github.com/kubernetes/test-infra/archive/6304ab49bc654eb0799d57ae2b076a246cb2051f.tar.gz"],
)

# TODO(fejta): create a test_infra_repositories and delete the below
## IMPLICIT test-infra repos

http_archive(
    name = "io_bazel_rules_k8s",
    sha256 = "91fef3e6054096a8947289ba0b6da3cba559ecb11c851d7bdfc9ca395b46d8d8",
    strip_prefix = "rules_k8s-0.1",
    urls = ["https://github.com/bazelbuild/rules_k8s/archive/v0.1.tar.gz"],
)

load("@io_bazel_rules_k8s//k8s:k8s.bzl", "k8s_repositories")

k8s_repositories()

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "aed1c249d4ec8f703edddf35cbe9dfaca0b5f5ea6e4cd9e83e99f3b0d1136c3d",
    strip_prefix = "rules_docker-0.7.0",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.7.0.tar.gz"],
)

load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_repositories = "repositories",
)

_go_repositories()

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load("@io_bazel_rules_docker//container:container.bzl", "container_pull")

http_archive(
    name = "com_google_protobuf",
    sha256 = "2ee9dcec820352671eb83e081295ba43f7a4157181dad549024d7070d079cf65",
    strip_prefix = "protobuf-3.9.0",
    urls = ["https://github.com/protocolbuffers/protobuf/archive/v3.9.0.tar.gz"],
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()
## END test-infra repos
