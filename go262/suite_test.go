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
	"fmt"
	"github.com/stvp/assert"
	"testing"
)

func TestEasy(t *testing.T) {
	g := RecursivelyWalk("./testdata_new")
	assert.NotEqual(t, g, nil, "didn't get a state")
	assert.NotEqual(t, g.rootSuite, nil, "didn't get a root suite")

	// make sure we have a single suite, and a single test
	assert.Equal(t, len(g.rootSuite.Suites), 1, "unexpected suites")
	assert.Equal(t, len(g.rootSuite.Tests), 1, "unexpected tests")

	// make sure the test is linked to us too
	tc := g.rootSuite.Tests[0]
	for _, suite := range tc.Suites {
		fmt.Printf("Suite %s\n", suite.PathName)
	}
	assert.Equal(t, len(tc.Suites), 1, "unexpected suites")
	assert.Equal(t, tc.Suites[0], g.rootSuite, "unexpected suite")
}

func BenchmarkDetermineRunJobs(b *testing.B) {
	g := RecursivelyWalk("./testdata_new")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.rootSuite.DetermineRunJobs()
	}
}
