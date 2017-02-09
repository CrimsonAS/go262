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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
)

type excludedCase struct {
	pathName string
}

func (global *GlobalState) removeExclusion(pathName string) {
	elist := global.excludeList
	for i, exc := range elist {
		if exc.pathName == pathName {
			elist = append(elist[:i], elist[i+1:]...)
		}
	}
	global.excludeList = elist
	global.writeExcludeList()
}

func (global *GlobalState) addExclusion(pathName string) {
	global.removeExclusion(pathName) // make sure to not duplicate
	global.excludeList = append(global.excludeList, excludedCase{pathName})
	global.writeExcludeList()
}

type exclusionSorter []excludedCase

func (a exclusionSorter) Len() int      { return len(a) }
func (a exclusionSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a exclusionSorter) Less(i, j int) bool {
	return a[i].pathName < a[j].pathName
}

func (global *GlobalState) writeExcludeList() {
	f, err := os.Create("TestExpectations")
	if err != nil {
		panic("Can't write expectations: " + err.Error())
	}
	defer f.Close()
	sort.Sort(exclusionSorter(global.excludeList))
	for _, exc := range global.excludeList {
		if strings.Contains(exc.pathName, " ") {
			panic("path contains space!")
		}
		f.WriteString(fmt.Sprintf("%s\n", exc.pathName))
	}
}

func (global *GlobalState) readExpectations() {
	// Read expectations data
	f, err := os.Open("TestExpectations")
	if err != nil {
		log.Printf("Can't read expectations (this might be OK): " + err.Error())
		return
	}
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	lines := bytes.Split(buf, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		global.excludeList = append(global.excludeList, excludedCase{string(line)})
	}
}

func (global *GlobalState) isExcluded(pathName string) bool {
	elist := global.excludeList
	for _, exc := range elist {
		if exc.pathName == pathName || strings.HasPrefix(pathName, exc.pathName) {
			return true
		}
	}

	return false
}

func (suite *TestSuite) SetExcluded(excluded bool) {
	if !excluded {
		suite.global.removeExclusion(suite.PathName + "/")
	} else {
		suite.global.addExclusion(suite.PathName + "/")
	}
}

func (suite *TestSuite) IsExcluded() bool {
	return suite.global.isExcluded(suite.PathName + "/")
}
