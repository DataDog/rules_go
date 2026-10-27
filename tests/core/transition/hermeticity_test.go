// Copyright 2021 The Bazel Authors. All rights reserved.
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

package hermeticity_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel_testing"
)

func TestMain(m *testing.M) {
	bazel_testing.TestMain(m, bazel_testing.Args{
		Main: `
-- BUILD.bazel --
load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_test")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@rules_proto//proto:defs.bzl", "proto_library")

go_binary(
    name = "main",
    srcs = [
        "main.go",
        ":gen_go",
    ],
    data = [":helper"],
    embedsrcs = [":helper"],
    cdeps = [":helper"],
    cgo = True,
    linkmode = "c-archive",
    gotags = ["foo"],
    deps = [":lib"],
)

go_library(
    name = "lib",
    srcs = [
        "lib.go",
        ":gen_indirect_go",
    ],
    importpath = "example.com/lib",
    data = [":indirect_helper"],
    embedsrcs = [":indirect_helper"],
    cdeps = [":indirect_helper"],
    cgo = True,
)

go_test(
    name = "main_test",
    srcs = [
        "main.go",
        ":gen_go",
    ],
    data = [":helper"],
    embedsrcs = [":helper"],
    cdeps = [":helper"],
    cgo = True,
    linkmode = "c-archive",
    gotags = ["foo"],
)

cc_library(
    name = "helper",
)

cc_library(
    name = "indirect_helper",
)

genrule(
    name = "gen_go",
    outs = ["gen.go"],
    tools = [":helper"],
    cmd = "# Not needed for bazel cquery",
)

genrule(
    name = "gen_indirect_go",
    outs = ["gen_indirect.go"],
    tools = [":indirect_helper"],
    cmd = "# Not needed for bazel cquery",
)

proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
)

go_proto_library(
    name = "foo_go_proto",
    importpath = "github.com/bazelbuild/rules_go/tests/core/transition/foo",
    proto = ":foo_proto",
)
-- main.go --
package main

func main() {}
-- lib.go --
-- foo.proto --
syntax = "proto3";

package tests.core.transition.foo;
option go_package = "github.com/bazelbuild/rules_go/tests/core/transition/foo";

message Foo {
  int64 value = 1;
}
`,
		ModuleFileSuffix: `
bazel_dep(name = "protobuf", version = "29.3", repo_name = "com_google_protobuf")
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
	})
}

func TestGoBinaryNonGoAttrsAreReset(t *testing.T) {
	assertDependsCleanlyOnWithFlags(
		t,
		"//:main",
		"//:helper",
		false)
}

func TestGoLibraryNonGoAttrsAreReset(t *testing.T) {
	assertDependsCleanlyOnWithFlags(
		t,
		"//:main",
		"//:indirect_helper",
		false)
}

func TestGoTestNonGoAttrsAreReset(t *testing.T) {
	assertDependsCleanlyOnWithFlags(
		t,
		"//:main_test",
		"//:helper",
		false)
}

func TestGoProtoLibraryToolAttrsAreReset(t *testing.T) {
	assertDependsCleanlyOnWithFlags(
		t,
		"//:foo_go_proto",
		"@com_google_protobuf//:protoc",
		// No dep with --incompatible_enable_proto_toolchain_resolution.
		true,
		"--@io_bazel_rules_go//go/config:static",
		"--@io_bazel_rules_go//go/config:msan",
		"--@io_bazel_rules_go//go/config:race",
		"--@io_bazel_rules_go//go/config:debug",
		"--@io_bazel_rules_go//go/config:linkmode=c-archive",
		"--@io_bazel_rules_go//go/config:tags=fake_tag",
	)
	assertDependsCleanlyOnWithFlags(
		t,
		"//:foo_go_proto",
		"@com_google_protobuf//:protoc",
		true,
		"--@io_bazel_rules_go//go/config:pure",
	)
}

func assertDependsCleanlyOnWithFlags(t *testing.T, targetA, targetB string, allowNoDep bool, flags ...string) {
	// Analyze the targets to ensure that MODULE.bazel.lock has been created, otherwise bazel config
	// will fail after the cquery command due to the Skyframe invalidation caused by a changed file.
	err := bazel_testing.RunBazel(append([]string{"build", targetA, targetB, "--nobuild"}, flags...)...)
	if err != nil {
		t.Fatalf("bazel build %s %s: %v", targetA, targetB, err)
	}
	query := fmt.Sprintf("deps(%s) intersect %s", targetA, targetB)
	out, err := bazel_testing.BazelOutput(append(
		[]string{
			"cquery",
			"--transitions=full",
			"--output=jsonproto",
			query,
		},
		flags...,
	)...,
	)
	if err != nil {
		t.Fatalf("bazel cquery '%s': %v", query, err)
	}
	cqueryOut := bytes.TrimSpace(out)
	configHashes := extractConfigHashes(t, cqueryOut)
	if len(configHashes) == 0 {
		if allowNoDep {
			return
		}
		t.Fatalf("%s does not depend on %s", targetA, targetB)
	}
	if len(configHashes) != 1 {
		differingGoOptions := getGoOptions(t, configHashes...)
		if len(differingGoOptions) != 0 {
			t.Fatalf(
				"%s depends on %s in multiple configs with these differences in rules_go options: %s",
				targetA,
				targetB,
				strings.Join(differingGoOptions, "\n"),
			)
		}
	}
	goOptions := getGoOptions(t, configHashes[0])
	if len(goOptions) != 0 {
		t.Fatalf(
			"%s depends on %s in a config with rules_go options: %s",
			targetA,
			targetB,
			strings.Join(goOptions, "\n"),
		)
	}
}

func extractConfigHashes(t *testing.T, rawJsonOut []byte) []string {
	var jsonOut bazelCqueryOutput
	err := json.Unmarshal(rawJsonOut, &jsonOut)
	if err != nil {
		t.Fatalf("Failed to decode bazel config JSON output %v: %q", err, string(rawJsonOut))
	}
	var hashes []string
	for _, result := range jsonOut.Results {
		hashes = append(hashes, result.Configuration.Checksum)
	}
	return hashes
}

func getGoOptions(t *testing.T, hashes ...string) []string {
	out, err := bazel_testing.BazelOutput(append([]string{"config", "--output=json"}, hashes...)...)
	if err != nil {
		t.Fatalf("bazel config %s: %v", strings.Join(hashes, " "), err)
	}
	rawJsonOut := bytes.TrimSpace(out)
	var jsonOut bazelConfigOutput
	err = json.Unmarshal(rawJsonOut, &jsonOut)
	if err != nil {
		t.Fatalf("Failed to decode bazel config JSON output %v: %q", err, string(rawJsonOut))
	}
	var differingGoOptions []string
	for _, fragment := range jsonOut.Fragments {
		if fragment.Name != starlarkOptionsFragment {
			continue
		}
		for key, value := range fragment.Options {
			if strings.HasPrefix(key, "@io_bazel_rules_go//") {
				differingGoOptions = append(differingGoOptions, fmt.Sprintf("%s=%s", key, value))
			}
		}
	}
	return differingGoOptions
}

const starlarkOptionsFragment = "user-defined"

type bazelConfigOutput struct {
	Fragments []struct {
		Name    string            `json:"name"`
		Options map[string]string `json:"options"`
	} `json:"fragmentOptions"`
}

type bazelCqueryOutput struct {
	Results []struct {
		Configuration struct {
			Checksum string `json:"checksum"`
		} `json:"configuration"`
	} `json:"results"`
}
