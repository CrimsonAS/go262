/*
 * Copyright (c) 2017 Crimson AS <info@crimson.no>
 * Author: Robin Burchell <robin.burchell@crimson.no>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package go262

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"
)

// An individual testcase (.js file)
type TestCase struct {
	// The full path to this test
	PathName string

	// The metadata of this test
	Metadata TestMetadata

	// The data of this test
	TestData string

	// All the suites associated with this test
	Suites []*TestSuite

	// The result of the last runs
	// RunType -> result
	// Beware, this is set from a different thread. Ensure you access it
	// under caseLock.
	lastResults map[string]*TestResult

	// Lock access to lastResults
	caseLock sync.Mutex

	global *GlobalState
}

// The in-line metadata related to this test. See the test262 documentation for
// details about what each of these are.
type TestMetadata struct {
	Description string
	Info        string
	Negative    struct {
		Phase string
		Type  string
	}
	EsId     string
	Es6Id    string
	Es5Id    string
	Includes []string
	Timeout  int
	Author   string
	Flags    []string
	Features []string
}

// Flags
const StrictFlag = "onlyStrict"
const NonStrictFlag = "noStrict"
const RawFlag = "raw"

// Negative.Phase
const EarlyPhase = "early"
const RuntimePhase = "runtime"

// Used to request a specific "job" for a test case. This is needed as tests can
// be run in either strict or non-strict environments (amongst possibly other
// things)
type TestJob struct {
	TestCase *TestCase

	// e.g. strict, non-strict
	RunType string
}

func (testcase *TestCase) SetExcluded(excluded bool) {
	if !excluded {
		testcase.global.removeExclusion(testcase.PathName)
	} else {
		testcase.global.addExclusion(testcase.PathName)
	}
}

func (testcase *TestCase) IsExcluded() bool {
	return testcase.global.isExcluded(testcase.PathName)
}

func (global *GlobalState) FetchTestcase(pathName string) *TestCase {
	pathName = path.Clean(pathName)
	return global.testMap[pathName]
}

func (testcase *TestCase) IsNegative() bool {
	if len(testcase.Metadata.Negative.Phase) > 0 || len(testcase.Metadata.Negative.Type) > 0 {
		return true
	}

	return false
}

func (suite *TestSuite) createTestCase(pathName string) {
	test := &TestCase{pathName,
		TestMetadata{},
		"",
		[]*TestSuite{},
		map[string]*TestResult{},
		sync.Mutex{},
		suite.global,
	}

	suite.global.testMap[pathName] = test

	// Add this test to the right suite
	suite.Tests = append(suite.Tests, test)

	// Find first alternative suite
	pathName = path.Dir(pathName)

	// Link to all other suites too for summarization purposes
	for ok := true; ok; ok = pathName != "." {
		suite := suite.global.FetchSuite(pathName)
		test.Suites = append(test.Suites, suite)
		pathName = path.Dir(pathName)
	}
}

func (testcase *TestCase) SuiteDir() string {
	return path.Dir(testcase.PathName)
}

func (testcase *TestCase) FileName() string {
	_, testName := path.Split(testcase.PathName)
	return testName
}

// Parse metadata out of this test case.
func (testcase *TestCase) Parse() {
	f, err := os.Open(testcase.PathName)
	if err != nil {
		panic(err)
	}

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	f.Close()
	testcase.ParseMetadata(string(bytes))
	testcase.verifyIncludes()
}

func (testcase *TestCase) HasFlag(str string) bool {
	for _, v := range testcase.Metadata.Flags {
		if v == str {
			return true
		}
	}

	return false
}

func (testcase *TestCase) GetSource(job *TestJob) string {
	// Get source
	if testcase.HasFlag(RawFlag) {
		return testcase.TestData
	}

	source := ""

	if job.RunType == "strict" {
		source = `"use strict";`
		source += "\nvar strict_mode = true;\n"
	} else if job.RunType == "nonstrict" {
		// Add a comment to get the line numbers to match
		source = `//"no strict";`
		source += "\nvar strict_mode = false;\n"
	} else {
		panic("Unknown job type " + job.RunType)
	}

	if testcase.Metadata.Negative.Phase == EarlyPhase {
		source += "throw 'Expected an early error, but code was executed.';\n"
	}

	// No need to check the include cache at this point. verifyIncludes
	// should have already caught any problem.
	_, inc := testcase.global.fetchFromIncludeCache("sta.js")
	source += inc
	_, inc = testcase.global.fetchFromIncludeCache("cth.js")
	source += inc
	_, inc = testcase.global.fetchFromIncludeCache("assert.js")
	source += inc

	// ###
	// if IsAsyncTest
	// read timer.js
	// doneprintHandle.js .replace('print', self.suite.print_handle

	// Specific test includes
	for _, inc := range testcase.Metadata.Includes {
		_, incsource := testcase.global.fetchFromIncludeCache(inc)
		source += incsource + "\n"
	}

	source += testcase.TestData

	return source
}

// Runs the job (in a blocking manner), and return a result.
func (testcase *TestCase) Run(job *TestJob) *TestResult {
	//fmt.Printf("Running %s\n", testcase.FileName())
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		panic("can't get tempfile! " + err.Error())
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(testcase.GetSource(job))); err != nil {
		panic("can't write tmpfile! " + err.Error())
	}
	if err := tmpfile.Close(); err != nil {
		panic("can't close tmpfile! " + err.Error())
	}

	// Run it
	startTime := time.Now()
	cmd := exec.Command("/Users/burchr/code/qt/qtbase/bin/qmljs", tmpfile.Name())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	//fmt.Printf("Done running %s\n", testcase.FileName())
	if err != nil {
		_, ok := err.(*exec.ExitError)
		if !ok {
			panic("Fatal error running: " + err.Error())
		}
	}

	tr := &TestResult{
		job,
		err == nil,
		stderr.String(),
		stdout.String(),
		time.Since(startTime),
	}

	testcase.caseLock.Lock()
	testcase.lastResults[tr.RunType] = tr
	testcase.caseLock.Unlock()

	return tr
}

func (testcase *TestCase) GetLastResultFor(runType string) *TestResult {
	testcase.caseLock.Lock()
	r := testcase.lastResults[runType]
	testcase.caseLock.Unlock()
	return r
}

// Create a number of TestJob instances for this TestCase.
func (testcase *TestCase) DetermineRunJobs() []*TestJob {
	onlyStrict := testcase.HasFlag(StrictFlag)
	noStrict := testcase.HasFlag(NonStrictFlag)
	//raw := testcase.HasFlag( RawFlag)
	//module := testcase.HasFlag( "module")
	//async := testcase.HasFlag( "async")

	var jobs []*TestJob

	if onlyStrict {
		jobs = append(jobs, &TestJob{testcase, "strict"})
	} else if noStrict {
		jobs = append(jobs, &TestJob{testcase, "nonstrict"})
	} else {
		jobs = append(jobs, &TestJob{testcase, "strict"})
		jobs = append(jobs, &TestJob{testcase, "nonstrict"})
	}

	return jobs
}

func (testcase *TestCase) HasRunType(runType string) bool {
	onlyStrict := testcase.HasFlag(StrictFlag)
	noStrict := testcase.HasFlag(NonStrictFlag)

	if noStrict && runType == "strict" {
		return false
	}

	if onlyStrict && runType != "strict" {
		return false
	}

	return true
}

const WillNotRunState = "wontrun"
const HasNotRunState = "hasntrun"
const PartialSuccessState = "mostlygood"
const SuccessState = "allgood"
const FailureState = "allbad"

func (testcase *TestCase) StateValue(runType string) string {
	// Test doesn't run in this mode
	if !testcase.HasRunType(runType) || testcase.IsExcluded() {
		return WillNotRunState
	}

	testcase.caseLock.Lock()
	res := testcase.lastResults[runType]
	testcase.caseLock.Unlock()
	if res == nil {
		return HasNotRunState
	}

	if res.IsSuccessful() {
		return SuccessState
	} else {
		return FailureState
	}
}
