# gazelle:repository_macro repos.bzl%go_repositories
workspace(name = "io_k8s_org")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "bazel_skylib",
    sha256 = "2ef429f5d7ce7111263289644d233707dba35e39696377ebab8b0bc701f7818e",
    urls = ["https://github.com/bazelbuild/bazel-skylib/releases/download/0.8.0/bazel-skylib.0.8.0.tar.gz"],
)

load("@bazel_skylib//lib:versions.bzl", "versions")

versions.check(minimum_bazel_version = "0.27.0")

http_archive(
    name = "io_k8s_repo_infra",
    sha256 = "f6d65480241ec0fd7a0d01f432938b97d7395aeb8eefbe859bb877c9b4eafa56",
    strip_prefix = "repo-infra-9f4571ad7242bf3ec4b47365062498c2528f9a5f",
    urls = ["https://github.com/kubernetes/repo-infra/archive/9f4571ad7242bf3ec4b47365062498c2528f9a5f.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "313f2c7a23fecc33023563f082f381a32b9b7254f727a7dd2d6380ccc6dfe09b",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.19.3/rules_go-0.19.3.tar.gz"],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "7fc87f4170011201b1690326e8c16c5d802836e3a0d617d8f75c3af2b23180c4",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.18.2/bazel-gazelle-0.18.2.tar.gz"],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

load("//:repos.bzl", "go_repositories")

go_repositories()

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
