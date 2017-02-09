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
	"io/ioutil"
	"log"
	"os"
	"path"
)

func (global *GlobalState) fetchFromIncludeCache(file string) (error, string) {
	if len(global.includeCache[file]) == 0 {
		return errors.New("Can't fetch include " + file), ""
	}
	return nil, global.includeCache[file]
}

func (global *GlobalState) walkHarnessIncludes(pathName string, info os.FileInfo, err error) error {
	_, name := path.Split(pathName)

	if info.IsDir() {
		return nil
	}

	bytes, err := ioutil.ReadFile(pathName)
	if err != nil {
		log.Fatalf("Can't read file! %s: %s", pathName, err.Error())
	}

	global.includeCache[name] = string(bytes)
	return nil
}

func init() {
}

func (testcase *TestCase) verifyIncludes() {
	for _, s := range testcase.Metadata.Includes {
		if err, _ := testcase.global.fetchFromIncludeCache(s); err != nil {
			log.Fatalf("Cannot find include %s\n", s)
		}
	}
}
