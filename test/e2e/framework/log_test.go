/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework_test

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"k8s.io/kubernetes/test/e2e/framework"
)

// The line number of the following code is checked in TestFailureOutput below.
// Be careful when moving it around or changing the import statements above.
// Here are some intentionally blank lines that can be removed to compensate
// for future additional import statements.
//
//
//
//
//

func runTests(t *testing.T, reporter ginkgo.Reporter) {
	// This source code line will be part of the stack dump comparison.
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Logging Suite", []ginkgo.Reporter{reporter})
}

var _ = ginkgo.Describe("log", func() {
	ginkgo.BeforeEach(func() {
		framework.Logf("before")
	})
	ginkgo.It("fails", func() {
		func() {
			framework.Failf("I'm failing.")
		}()
	})
	ginkgo.It("asserts", func() {
		framework.ExpectEqual(false, true, "false is never true")
	})
	ginkgo.It("error", func() {
		err := errors.New("an error with a long, useless description")
		framework.ExpectNoError(err, "hard-coded error")
	})
	ginkgo.It("equal", func() {
		framework.ExpectEqual(0, 1, "of course it's not equal...")
	})
	ginkgo.AfterEach(func() {
		framework.Logf("after")
		framework.ExpectEqual(true, false, "true is never false either")
	})
})

func TestFailureOutput(t *testing.T) {
	// Run the Ginkgo suite with output collected by a custom
	// reporter in adddition to the default one. To see what the full
	// Ginkgo report looks like, run this test with "go test -v".
	config.DefaultReporterConfig.FullTrace = true
	gomega.RegisterFailHandler(framework.Fail)
	fakeT := &testing.T{}
	reporter := reporters.NewFakeReporter()
	runTests(fakeT, reporter)

	// Now check the output.
	actual := normalizeReport(*reporter)

	// output from AfterEach
	commonOutput := `

INFO: after
FAIL: true is never false either
Expected
    <bool>: true
to equal
    <bool>: false

Full Stack Trace
k8s.io/kubernetes/test/e2e/framework_test.glob..func1.6()
	log_test.go:71
k8s.io/kubernetes/test/e2e/framework_test.runTests()
	log_test.go:47

`

	// Sorted by name!
	expected := suiteResults{
		testResult{
			name: "[Top Level] log asserts",
			output: `INFO: before
FAIL: false is never true
Expected
    <bool>: false
to equal
    <bool>: true

Full Stack Trace
k8s.io/kubernetes/test/e2e/framework_test.glob..func1.3()
	log_test.go:60
k8s.io/kubernetes/test/e2e/framework_test.runTests()
	log_test.go:47` + commonOutput,
			failure: `false is never true
Expected
    <bool>: false
to equal
    <bool>: true`,
			stack: `k8s.io/kubernetes/test/e2e/framework_test.glob..func1.3()
	log_test.go:60
k8s.io/kubernetes/test/e2e/framework_test.runTests()
	log_test.go:47
`,
		},
		testResult{
			name: "[Top Level] log equal",
			output: `INFO: before
FAIL: of course it's not equal...
Expected
    <int>: 0
to equal
    <int>: 1

Full Stack Trace
k8s.io/kubernetes/test/e2e/framework_test.glob..func1.5()
	log_test.go:67
k8s.io/kubernetes/test/e2e/framework_test.runTests()
	log_test.go:47` + commonOutput,
			failure: `of course it's not equal...
Expected
    <int>: 0
to equal
    <int>: 1`,
			stack: `k8s.io/kubernetes/test/e2e/framework_test.glob..func1.5()
	log_test.go:67
k8s.io/kubernetes/test/e2e/framework_test.runTests()
	log_test.go:47
`,
		},
		testResult{
			name: "[Top Level] log error",
			output: `INFO: before
FAIL: hard-coded error
Unexpected error:
    <*errors.errorString>: {
        s: "an error with a long, useless description",
    }
    an error with a long, useless description
occurred

Full Stack Trace
k8s.io/kubernetes/test/e2e/framework_test.glob..func1.4()
	log_test.go:64
k8s.io/kubernetes/test/e2e/framework_test.runTests()
	log_test.go:47` + commonOutput,
			failure: `hard-coded error
Unexpected error:
    <*errors.errorString>: {
        s: "an error with a long, useless description",
    }
    an error with a long, useless description
occurred`,
			stack: `k8s.io/kubernetes/test/e2e/framework_test.glob..func1.4()
	log_test.go:64
k8s.io/kubernetes/test/e2e/framework_test.runTests()
	log_test.go:47
`,
		},
		testResult{
			name: "[Top Level] log fails",
			output: `INFO: before
FAIL: I'm failing.

Full Stack Trace
k8s.io/kubernetes/test/e2e/framework_test.glob..func1.2()
	log_test.go:57
k8s.io/kubernetes/test/e2e/framework_test.runTests()
	log_test.go:47` + commonOutput,
			failure: "I'm failing.",
			stack: `k8s.io/kubernetes/test/e2e/framework_test.glob..func1.2()
	log_test.go:57
k8s.io/kubernetes/test/e2e/framework_test.runTests()
	log_test.go:47
`,
		},
	}
	// assert.Equal prints a useful diff if the slices are not
	// equal. However, the diff does not show changes inside the
	// strings. Therefore we also compare the individual fields.
	if !assert.Equal(t, expected, actual) {
		for i := 0; i < len(expected) && i < len(actual); i++ {
			assert.Equal(t, expected[i].output, actual[i].output, "output from test #%d: %s", i, expected[i].name)
			assert.Equal(t, expected[i].stack, actual[i].stack, "stack from test #%d: %s", i, expected[i].name)
		}
	}
}

type testResult struct {
	name string
	// output written to GinkgoWriter during test.
	output string
	// failure is SpecSummary.Failure.Message with varying parts stripped.
	failure string
	// stack is a normalized version (just file names, function parametes stripped) of
	// Ginkgo's FullStackTrace of a failure. Empty if no failure.
	stack string
}

type suiteResults []testResult

func normalizeReport(report reporters.FakeReporter) suiteResults {
	var results suiteResults
	for _, spec := range report.SpecSummaries {
		results = append(results, testResult{
			name:    strings.Join(spec.ComponentTexts, " "),
			output:  normalizeLocation(stripAddresses(stripTimes(spec.CapturedOutput))),
			failure: stripAddresses(stripTimes(spec.Failure.Message)),
			stack:   normalizeLocation(spec.Failure.Location.FullStackTrace),
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return strings.Compare(results[i].name, results[j].name) < 0
	})
	return results
}

// timePrefix matches "Jul 17 08:08:25.950: " at the beginning of each line.
var timePrefix = regexp.MustCompile(`(?m)^[[:alpha:]]{3} +[[:digit:]]{1,2} +[[:digit:]]{2}:[[:digit:]]{2}:[[:digit:]]{2}.[[:digit:]]{3}: `)

func stripTimes(in string) string {
	return timePrefix.ReplaceAllString(in, "")
}

// instanceAddr matches " | 0xc0003dec60>"
var instanceAddr = regexp.MustCompile(` \| 0x[0-9a-fA-F]+>`)

func stripAddresses(in string) string {
	return instanceAddr.ReplaceAllString(in, ">")
}

// stackLocation matches "<some path>/<file>.go:75 +0x1f1" after a slash (built
// locally) or one of a few relative paths (built in the Kubernetes CI).
var stackLocation = regexp.MustCompile(`(?:/|vendor/|test/|GOROOT/).*/([[:^space:]]+.go:[[:digit:]]+)( \+0x[0-9a-fA-F]+)?`)

// functionArgs matches "<function name>(...)".
var functionArgs = regexp.MustCompile(`([[:alpha:]]+)\(.*\)`)

// testFailureOutput matches TestFailureOutput() and its source followed by additional stack entries:
//
// k8s.io/kubernetes/test/e2e/framework_test.TestFailureOutput(0xc000558800)
//	/nvme/gopath/src/k8s.io/kubernetes/test/e2e/framework/log/log_test.go:73 +0x1c9
// testing.tRunner(0xc000558800, 0x1af2848)
// 	/nvme/gopath/go/src/testing/testing.go:865 +0xc0
// created by testing.(*T).Run
//	/nvme/gopath/go/src/testing/testing.go:916 +0x35a
var testFailureOutput = regexp.MustCompile(`(?m)^k8s.io/kubernetes/test/e2e/framework_test\.TestFailureOutput\(.*\n\t.*(\n.*\n\t.*)*`)

// normalizeLocation removes path prefix and function parameters and certain stack entries
// that we don't care about.
func normalizeLocation(in string) string {
	out := in
	out = stackLocation.ReplaceAllString(out, "$1")
	out = functionArgs.ReplaceAllString(out, "$1()")
	out = testFailureOutput.ReplaceAllString(out, "")
	return out
}
