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
	"errors"
	"regexp"
)

// For YAML parsing out of the testcases.
// Thanks to parseTestRecord.py.
// Copyright 2011 by Google, Inc.  All rights reserved.
const headerPatternStr = "(?:(?:\\s*\\/\\/.*)?\\s*\\n)*"
const captureCommentPatternStr = "\\/\\*\\*?((?:\\s|\\S)*?)\\*\\/\\s*\\n"
const anyPatternStr = "(?:\\s|\\S)*"

var headerPattern = regexp.MustCompile("^" + headerPatternStr)
var testRecordPattern = regexp.MustCompile("^(" + headerPatternStr +
	")(?:" + captureCommentPatternStr +
	")?(" + anyPatternStr +
	")$")

var stars = regexp.MustCompile("\\s*\\n\\s*\\*\\s?")
var atattrs = regexp.MustCompile("\\s*\\n\\s*\\*\\s*@")

var yamlPattern = regexp.MustCompile("---((?:\\s|\\S)*)---")
var newlinePattern = regexp.MustCompile("\\n")

func matchParts(src string) ([]string, error) {
	matches := testRecordPattern.FindStringSubmatch(src)
	if matches == nil {
		return nil, errors.New("can't find matchParts")
	}
	return matches, nil
}

var noYamlErr = errors.New("no YAML found")

func parseYamlAttr(text string) (error, TestMetadata) {
	matches := yamlPattern.FindStringSubmatch(text)
	if matches == nil {
		return noYamlErr, TestMetadata{}
	}

	return load(string(matches[1]))
}

func (testcase *TestCase) ParseMetadata(contents string) {
	match, err := matchParts(contents)
	if err != nil {
		panic("boom, " + testcase.PathName + " -- " + err.Error())
	}

	// needed?
	//testcase.TestRecord["header"] = strings.TrimSpace(match[1])
	testcase.TestData = match[3]

	attrs := match[2]
	if len(attrs) > 0 {
		err, meta := parseYamlAttr(attrs)
		if err != nil {
			if err == noYamlErr {
				panic("old attr support needed")
			} else {
				panic("boom! " + err.Error())
			}
		} else {
			testcase.Metadata = meta
		}
	}

	// Validate it
	if len(testcase.Metadata.Negative.Phase) > 0 {
		if testcase.Metadata.Negative.Phase != EarlyPhase &&
			testcase.Metadata.Negative.Phase != RuntimePhase {
			panic("Invalid testcase. Can't have a negative phase of " + testcase.Metadata.Negative.Phase)
		}
	}

	if testcase.HasFlag(RawFlag) {
		if testcase.HasFlag(StrictFlag) || testcase.HasFlag(NonStrictFlag) {
			panic("Invalid testcase. Can't be raw and strict or nonStrict")
		}

		if len(testcase.Metadata.Includes) > 0 {
			panic("Invalid testcase. Can't be raw and have includes")

		}
	}
}
