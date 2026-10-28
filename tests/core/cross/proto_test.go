// Copyright 2019 The Bazel Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proto_test

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel_testing"
)

var testArgs = bazel_testing.Args{
	ModuleFileSuffix: `
bazel_dep(name = "rules_proto", version = "7.1.0")
bazel_dep(name = "toolchains_protoc", version = "0.3.7")
`,
	WorkspacePrefix: `
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Bazel 8's bazel_tools requires a newer rules_cc than the one it bundles for WORKSPACE builds.
http_archive(
    name = "rules_cc",
    sha256 = "abc605dd850f813bb37004b77db20106a19311a96b2da1c92b789da529d28fe1",
    strip_prefix = "rules_cc-0.0.17",
    urls = ["https://github.com/bazelbuild/rules_cc/releases/download/0.0.17/rules_cc-0.0.17.tar.gz"],
)

# Bazel 8 injects loads from @rules_java into the WORKSPACE suffix.
http_archive(
    name = "rules_java",
    sha256 = "6d8c6d5cd86fed031ee48424f238fa35f33abc9921fd97dd4ae1119a29fc807f",
    urls = ["https://github.com/bazelbuild/rules_java/releases/download/8.6.3/rules_java-8.6.3.tar.gz"],
)

load("@rules_java//java:rules_java_deps.bzl", "compatibility_proxy_repo")

compatibility_proxy_repo()

# protobuf's bazel rules need a newer bazel_skylib than go_rules_dependencies() declares.
http_archive(
    name = "bazel_skylib",
    sha256 = "bc283cdfcd526a52c3201279cda4bc298652efa898b10b4db0837dc51652756f",
    urls = ["https://github.com/bazelbuild/bazel-skylib/releases/download/1.7.1/bazel-skylib-1.7.1.tar.gz"],
)

# The non-polyfill version of this is needed by rules_proto below.
http_archive(
    name = "bazel_features",
    sha256 = "d7787da289a7fb497352211ad200ec9f698822a9e0757a4976fd9f713ff372b3",
    strip_prefix = "bazel_features-1.9.1",
    url = "https://github.com/bazel-contrib/bazel_features/releases/download/v1.9.1/bazel_features-v1.9.1.tar.gz",
)

load("@bazel_features//:deps.bzl", "bazel_features_deps")

bazel_features_deps()
`,
	WorkspaceSuffix: `
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "com_google_protobuf",
    sha256 = "008a11cc56f9b96679b4c285fd05f46d317d685be3ab524b2a310be0fbad987e",
    strip_prefix = "protobuf-29.3",
    urls = ["https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protobuf-29.3.tar.gz"],
)

# protobuf_deps() would otherwise pull in a rules_python that is too old for Bazel 8.
http_archive(
    name = "rules_python",
    sha256 = "690e0141724abb568267e003c7b6d9a54925df40c275a870a4d934161dc9dd53",
    strip_prefix = "rules_python-0.40.0",
    url = "https://github.com/bazelbuild/rules_python/releases/download/0.40.0/rules_python-0.40.0.tar.gz",
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

http_archive(
    name = "rules_proto",
    sha256 = "14a225870ab4e91869652cfd69ef2028277fc1dc4910d65d353b62d6e0ae21f4",
    strip_prefix = "rules_proto-7.1.0",
    url = "https://github.com/bazelbuild/rules_proto/releases/download/7.1.0/rules_proto-7.1.0.tar.gz",
)

load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies")
rules_proto_dependencies()

load("@rules_proto//proto:toolchains.bzl", "rules_proto_toolchains")
rules_proto_toolchains()
`,
	Main: `
-- BUILD.bazel --
load("@rules_proto//proto:defs.bzl", "proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@io_bazel_rules_go//go:def.bzl", "go_binary")

proto_library(
    name = "cross_proto",
    srcs = ["cross.proto"],
)

go_proto_library(
    name = "cross_go_proto",
    importpath = "github.com/bazelbuild/rules_go/tests/core/cross",
    protos = [":cross_proto"],
)

go_binary(
    name = "use_bin",
    srcs = ["use.go"],
    deps = [":cross_go_proto"],
    goos = "linux",
    goarch = "386",
)

go_binary(
    name = "use_shared",
    srcs = ["use.go"],
    deps = [":cross_go_proto"],
    linkmode = "c-shared",
)

-- cross.proto --
syntax = "proto3";

package cross;

option go_package = "github.com/bazelbuild/rules_go/tests/core/cross";

message Foo {
  int64 x = 1;
}

-- use.go --
package main

import _ "github.com/bazelbuild/rules_go/tests/core/cross"

func main() {}
`,
}

func TestMain(m *testing.M) {
	bazel_testing.TestMain(m, testArgs)
}

func TestCmdLine(t *testing.T) {
	args := []string{
		"build",
		"--platforms=@io_bazel_rules_go//go/toolchain:linux_386",
		":cross_go_proto",
	}
	if err := bazel_testing.RunBazel(args...); err != nil {
		t.Fatal(err)
	}
}

func TestTargets(t *testing.T) {
	for _, target := range []string{"//:use_bin", "//:use_shared"} {
		if err := bazel_testing.RunBazel("build", target); err != nil {
			t.Errorf("building target %s: %v", target, err)
		}
	}
}
