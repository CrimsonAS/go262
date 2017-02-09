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
	"os"
	"path"
	"path/filepath"
)

// A collection of test cases (.js files)
type TestSuite struct {
	// The full path to this suite (note that a suite is a directory).
	PathName string

	// The tests that belong to this suite
	Tests []*TestCase

	// Suites under this suite
	Suites []*TestSuite

	global *GlobalState
}

// The root of all evil
type GlobalState struct {
	// Container of all suites.
	// map[path] -> TestSuite
	suiteMap map[string]*TestSuite

	// Container of all tests.
	// map[path] -> TestCase
	testMap map[string]*TestCase

	// The top-level directory "suite"
	rootSuite *TestSuite

	// Harness include cache.
	// filename -> file content
	includeCache map[string]string

	// List of excluded test paths
	excludeList []excludedCase
}

// Create all test cases and suites for a given path, returning the global state
// for use elsewhere (e.g. Go262Web)
func RecursivelyWalk(pathName string) *GlobalState {
	state := &GlobalState{
		make(map[string]*TestSuite),
		make(map[string]*TestCase),
		nil,
		make(map[string]string),
		nil,
	}

	state.readExpectations()

	filepath.Walk("./harness", state.walkHarnessIncludes)

	walkWrapper := func(pathName string, info os.FileInfo, err error) error {
		return walkSuitesAndTests(state, pathName, info, err)
	}

	filepath.Walk(pathName, walkWrapper)

	suite := state.rootSuite
	suite.parseRecursive()
	return state
}

func (suite *TestSuite) parseRecursive() {
	for _, t := range suite.Tests {
		t.Parse()
	}

	for _, v := range suite.Suites {
		v.parseRecursive()
	}
}

// Walks a directory tree, creating TestCase and TestSuite instances, ready for
// further processing
func walkSuitesAndTests(global *GlobalState, pathName string, info os.FileInfo, err error) error {
	if err != nil {
		panic("can't walk " + pathName + ": " + err.Error())
	}

	if info.IsDir() {
		global.createSuiteIfNeeded(pathName)
		return nil
	} else {
		// Link the test to the primary suite
		suitePath := path.Dir(pathName)
		suite := global.FetchSuite(suitePath)
		if suite == nil {
			panic("Can't find suite!?")
		}
		suite.createTestCase(pathName)
	}
	return nil
}

func (global *GlobalState) RootSuite() *TestSuite {
	return global.rootSuite
}

func (global *GlobalState) FetchSuite(pathName string) *TestSuite {
	pathName = path.Clean(pathName)
	return global.suiteMap[pathName]
}

// Creates a suite, or fetches a previously created suite for pathName if one
// was already created.
func (global *GlobalState) createSuiteIfNeeded(pathName string) *TestSuite {
	pathName = path.Clean(pathName)
	if pathName == "." {
		panic("not right")
	}
	suite := global.suiteMap[pathName]
	if suite != nil {
		return suite
	}
	suite = &TestSuite{PathName: pathName, global: global}
	global.suiteMap[pathName] = suite
	//fmt.Printf("Creating new suite %s\n", pathName)
	if global.rootSuite == nil {
		global.rootSuite = suite
	} else {
		// Find the suite to parent this suite to
		parent := global.FetchSuite(path.Dir(pathName))
		parent.Suites = append(parent.Suites, suite)
		//fmt.Printf("Parented %s to %s\n", suite.PathName, parent.PathName)
	}
	return suite
}

// Returns a list of all jobs in this suite, recursively.
func (suite *TestSuite) DetermineRunJobs() []*TestJob {
	var jobs []*TestJob

	for _, test := range suite.Tests {
		jobs = append(jobs, test.DetermineRunJobs()...)
	}

	for _, child := range suite.Suites {
		jobs = append(jobs, child.DetermineRunJobs()...)
	}

	return jobs
}

type SuiteResults struct {
	// Total tests for a type (that are valid, and not excluded)
	TotalCounts map[string]float64

	// Total tests for a type that were successful in the last run
	SuccessCounts map[string]float64

	// Total tests that are excluded
	ExcludedCounts map[string]float64
}

// Get results for this suite
func (suite *TestSuite) CalculateResults() SuiteResults {
	r := SuiteResults{
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
	}

	for _, test := range suite.Tests {
		calcForType := func(runType string, state string) {
			if state == WillNotRunState {
				r.ExcludedCounts[runType] += 1
				return
			}

			r.TotalCounts[runType] += 1

			if state == SuccessState {
				r.SuccessCounts[runType] += 1
			}
		}

		calcForType("strict", test.StateValue("strict"))
		calcForType("nonstrict", test.StateValue("nonstrict"))
	}

	return r
}

// ### track whether or not we have actually run tests, and return
// HasNotRunState if we haven't run them
func (suite *TestSuite) StateValue(runType string) string {
	r := suite.CalculateResults()
	if r.TotalCounts[runType] == 0 {
		if r.ExcludedCounts[runType] > 0 {
			return WillNotRunState
		}
		return HasNotRunState
	}

	if r.SuccessCounts[runType] == r.TotalCounts[runType] {
		return SuccessState
	} else if r.SuccessCounts[runType] >= r.TotalCounts[runType]/2 {
		return PartialSuccessState
	}

	return FailureState
}
