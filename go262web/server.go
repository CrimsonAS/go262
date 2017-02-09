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

package go262web

import (
	Go262 "../go262"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"path"
)

// ### rewrite this stuff using templates
// "text/template"
//tsource := `
//<h1><a href="{{.PathName}}">{{.PathName}}</a></h1>
//`
//temp := template.New("Range Example")
//temp.Parse(tsource)
//temp.Execute(os.Stdout, suite)

func headerBreadcrumbSuiteLink(suite *Go262.TestSuite, finalPart string) string {
	buf := "<h1>"
	buf += breadcrumbSuiteLink(suite, finalPart)
	buf += "</h1>"
	return buf
}

// Make a trail of breadcrumb links pointing to the suite
func breadcrumbSuiteLink(suite *Go262.TestSuite, finalPart string) string {
	buf := ""
	pathBit := suite.PathName
	var pathParts []string

	for pathBit != "." {
		_, tmp := path.Split(pathBit)
		pathParts = append(pathParts, tmp)
		pathBit = path.Dir(pathBit)
	}

	prevParts := ""
	for i := len(pathParts) - 1; i >= 0; i-- {
		prevParts += "/" + pathParts[i]
		buf += fmt.Sprintf(`<a href="/suite%s">/%s</a>`, prevParts, pathParts[i])
	}

	buf += finalPart
	return buf
}

func summarizeSuite(suite *Go262.TestSuite) string {
	r := suite.CalculateResults()
	buf := ""

	strictCountPerc := ""
	if r.TotalCounts["strict"] > 0 {
		succ := r.SuccessCounts["strict"]
		tot := r.TotalCounts["strict"]
		strictCountPerc = fmt.Sprintf("%.2f%% (%d of %d)", succ/tot*100, int(succ), int(tot))
	}
	nonStrictCountPerc := ""
	if r.TotalCounts["nonstrict"] > 0 {
		succ := r.SuccessCounts["nonstrict"]
		tot := r.TotalCounts["nonstrict"]
		nonStrictCountPerc = fmt.Sprintf("%.2f%% (%d of %d)", succ/tot*100, int(succ), int(tot))
	}
	presentState := func(state string) string {
		switch state {
		case Go262.WillNotRunState:
			return "black"
		case Go262.HasNotRunState:
			return ""
		case Go262.PartialSuccessState:
			return "yellow"
		case Go262.SuccessState:
			return "green"
		case Go262.FailureState:
			return "red"
		}
		return "blue"
	}

	if len(suite.Tests) > 0 {
		buf += "<tr>"
		buf += fmt.Sprintf(`<td>%s</td>`, breadcrumbSuiteLink(suite, ""))
		buf += fmt.Sprintf(`<td bgcolor="%s">%s</td>`, presentState(suite.StateValue("strict")), strictCountPerc)
		buf += fmt.Sprintf(`<td bgcolor="%s">%s</td>`, presentState(suite.StateValue("nonstrict")), nonStrictCountPerc)
		buf += "</tr>"
	}

	for _, child := range suite.Suites {
		buf += summarizeSuite(child)
	}

	return buf
}

func summarizeSuitesAndEverything(suite *Go262.TestSuite) string {
	buf := ""
	r := suite.CalculateResults()
	for _, test := range suite.Tests {
		buf += "<tr>"
		buf += fmt.Sprintf(`<td><a href="/test/%s" title="%ss">%s</a></td>`, test.PathName, test.Metadata.Description, test.FileName())

		presentState := func(state string) (string, string) {
			switch state {
			case Go262.WillNotRunState:
				return "black", ""
			case Go262.HasNotRunState:
				return "", ""
			case Go262.SuccessState:
				return "green", "true"
			case Go262.FailureState:
				return "red", "false"
			}
			return "blue", "WTF"
		}

		strictCol, strictSuccess := presentState(test.StateValue("strict"))
		nonStrictCol, nonStrictSuccess := presentState(test.StateValue("nonstrict"))

		buf += fmt.Sprintf(`<td bgcolor="%s">%s</td>`, strictCol, strictSuccess)
		buf += fmt.Sprintf(`<td bgcolor="%s">%s</td>`, nonStrictCol, nonStrictSuccess)
		buf += "</tr>"
	}

	buf += `<tr><th>Test</th><th>Pass Strict</th><th>Pass Nonstrict</th></tr>`
	buf += fmt.Sprintf(`<tr><th></th><th>%.2f%%</th><th>%.2f%%</th></tr>`, r.SuccessCounts["strict"]/r.TotalCounts["strict"]*100, r.SuccessCounts["nonstrict"]/r.TotalCounts["nonstrict"]*100)

	return buf
}

// Returns HTML for an overview of a whole suite (and any children suites), recursively.
func printSuite(suite *Go262.TestSuite, printHeaderIfEmpty bool) string {
	buf := ""

	if printHeaderIfEmpty || len(suite.Tests) > 0 {
		runPart := fmt.Sprintf(` - <a href="/run/%s">Run</a>`, suite.PathName)
		if !suite.IsExcluded() {
			runPart += fmt.Sprintf(` - <a href="/exclude/true/%s">Exclude</a>`, suite.PathName)
		} else {
			runPart += fmt.Sprintf(` - <a href="/exclude/false/%s">Unexclude</a>`, suite.PathName)
		}
		buf += headerBreadcrumbSuiteLink(suite, runPart)
	}

	if printHeaderIfEmpty && len(suite.Tests) == 0 {
		// Summarize all suites in here recursively
		buf += `<table border="1"><th>Suite</th><th>Pass Strict</th><th>Pass Nonstrict</th>`
		buf += summarizeSuite(suite)
		buf += `</table>`
	} else if len(suite.Tests) > 0 {
		buf += `<table border="1"><th>Test</th><th>Pass Strict</th><th>Pass Nonstrict</th>`
		buf += summarizeSuitesAndEverything(suite)
		buf += "</table>"

		for _, child := range suite.Suites {
			buf += printSuite(child, false)
		}
	}

	return buf
}

// / handler
func indexHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, printSuite(globalState.RootSuite(), true))
}

// /suite/<path> handler
func suiteShowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["path"]
	suite := globalState.FetchSuite(name)
	if suite == nil {
		errorHandler(w, r, http.StatusNotFound)
		return
	}

	io.WriteString(w, printSuite(suite, true))
}

// Prints information about a specific test
func printTest(test *Go262.TestCase) []byte {
	s := headerBreadcrumbSuiteLink(test.Suites[0], "/"+test.FileName())

	s += fmt.Sprintf("<b>Description</b>: %s<br>", test.Metadata.Description)
	s += fmt.Sprintf("<b>Info</b>: %s<br>", test.Metadata.Info)
	s += fmt.Sprintf("<b>Negative</b>: %s %s<br>", test.Metadata.Negative.Phase, test.Metadata.Negative.Type)
	s += fmt.Sprintf("<b>EsId</b>: %s<br>", test.Metadata.EsId)
	s += fmt.Sprintf("<b>Es6Id</b>: %s<br>", test.Metadata.Es6Id)
	s += fmt.Sprintf("<b>Es5Id</b>: %s<br>", test.Metadata.Es5Id)
	s += fmt.Sprintf("<b>Timeout</b>: %d<br>", test.Metadata.Timeout)
	s += fmt.Sprintf("<b>Author</b>: %s<br>", test.Metadata.Author)
	s += fmt.Sprintf("<b>Flags</b>: %s<br>", test.Metadata.Flags)
	s += fmt.Sprintf("<b>Features</b>: %s<br>", test.Metadata.Features)
	s += fmt.Sprintf(`<b>View Full</b>: <a href="/read/strict/%s">View Strict</a> <a href="/read/nonstrict/%s">View Non-strict</a><br>`, test.PathName, test.PathName)
	s += fmt.Sprintf(`<b>View Last Strict Logs</b>: <a href="/logs/strict/stderr/%s">Stderr</a> <a href="/logs/strict/stdout/%s">Stdout</a><br>`, test.PathName, test.PathName)
	s += fmt.Sprintf(`<b>View Last NonStrict Logs</b>: <a href="/logs/nonstrict/stderr/%s">Stderr</a> <a href="/logs/nonstrict/stdout/%s">Stdout</a><br>`, test.PathName, test.PathName)

	if test.IsExcluded() {
		s += fmt.Sprintf(`<b>Unexclude</b>: <a href="/exclude/false/%s">Unexclude</a><br>`, test.PathName)
	} else {
		s += fmt.Sprintf(`<b>Exclude</b>: <a href="/exclude/true/%s">Exclude</a><br>`, test.PathName)
	}

	s += fmt.Sprintf(`<b>Run</b>: <a href="/run/%s">Run</a><br>`, test.PathName)
	s += fmt.Sprintf("<pre>%s</pre>", test.TestData)
	// flags?
	// features?
	return []byte(s)
}

// /test/<path> handler
func testShowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["path"]
	test := globalState.FetchTestcase(name)
	if test == nil {
		errorHandler(w, r, http.StatusNotFound)
		return
	}

	w.Write(printTest(test))
}

var testRunnerPool = Go262.NewWorkerPool()

func runTestJobs(w http.ResponseWriter, jobs []*Go262.TestJob) {
	notify := w.(http.CloseNotifier).CloseNotify()

	// Create a new queue of jobs to run, push our jobs to it.
	q := Go262.NewJobQueue(testRunnerPool)
	go q.SendJobs(jobs)

	go func() {
		// If the HTTP connection goes away, cancel the running queue
		// ### I think this goroutine can leak if we return "normally"? :/
		<-notify
		q.Cancel()
	}()

	// Read all finished results from the queue, and process them
	for result := range q.ResultChannel {
		io.WriteString(w, fmt.Sprintf("Job %s(type: %s) finished in %s, success: %t\n", result.TestCase.FileName(), result.RunType, result.ExecutionDuration.String(), result.IsSuccessful()))
		if !result.IsSuccessful() {
			//if result.TestCase.IsNegative() {
			//	io.WriteString(w, fmt.Sprintf("### expected to fail in %s (with type %s), but didn't\n", result.TestCase.Metadata.Negative.Phase, result.TestCase.Metadata.Negative.Type))
			//}
			//io.WriteString(w, fmt.Sprintf("=== stderr ===\n%s\n", result.StderrOutput))
			//io.WriteString(w, fmt.Sprintf("=== stdout ===\n%s\n", result.StdoutOutput))
		}
	}
}

// /run/<path> handler
func runTestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["path"]
	suite := globalState.FetchSuite(name)
	if suite != nil {
		jobs := suite.DetermineRunJobs()
		runTestJobs(w, jobs)
		return
	} else {
		test := globalState.FetchTestcase(name)
		if test == nil {
			errorHandler(w, r, http.StatusNotFound)
			return
		}

		// Determine the jobs to send to the workers, and send them
		jobs := test.DetermineRunJobs()
		runTestJobs(w, jobs)
	}
}

// /read/{runtype}/<path>
func readCodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runType := vars["runtype"]
	name := vars["path"]

	test := globalState.FetchTestcase(name)
	if test == nil {
		errorHandler(w, r, http.StatusNotFound)
		return
	}

	j := Go262.TestJob{
		test,
		runType,
	}

	source := test.GetSource(&j)
	io.WriteString(w, source)
}

// Sends customized error pages
func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		fmt.Fprintf(w, "<h1>404</h1>\nCan't find that.")
	}
}

func logReq(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("B: %p %s %s\n", r, r.RemoteAddr, r.URL)
		fn(w, r)
		fmt.Printf("E: %p %s %s\n", r, r.RemoteAddr, r.URL)
	}
}

// /logs/{runtype}/{type}/<path>
func readLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runType := vars["runtype"]
	logType := vars["type"]
	name := vars["path"]

	if logType != "stderr" && logType != "stdout" {
		errorHandler(w, r, http.StatusNotFound)
		return
	}

	test := globalState.FetchTestcase(name)
	if test == nil {
		errorHandler(w, r, http.StatusNotFound)
		return
	}

	var res *Go262.TestResult = test.GetLastResultFor(runType)
	if res == nil {
		errorHandler(w, r, http.StatusNotFound)
		return
	}

	if logType == "stderr" {
		io.WriteString(w, res.StderrOutput)
	} else {
		io.WriteString(w, res.StdoutOutput)
	}
}

/*
// /expectedfail/{runtype}/<path>
func setExpectedFail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runType := vars["runtype"]
	name := vars["path"]

	suite := globalState.FetchSuite(name)
	if suite != nil {
		suite.SetExpectedFail(runType, true)
		return
	} else {
		test := globalState.FetchTestcase(name)
		if test == nil {
			errorHandler(w, r, http.StatusNotFound)
			return
		}

		test.SetExpectedFail(runType, true)
	}
}
*/

// /exclude/{truefalse}/<path>
func setExcludedHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["path"]
	truefalse := vars["truefalse"]

	excluded := true
	if truefalse == "true" {
		excluded = true
	} else if truefalse == "false" {
		excluded = false
	} else {
		errorHandler(w, r, http.StatusNotFound)
		return
	}

	suite := globalState.FetchSuite(name)
	if suite != nil {
		if excluded {
			io.WriteString(w, "Excluded OK")
		} else {
			io.WriteString(w, "Unexcluded OK")
		}
		suite.SetExcluded(excluded)
		return
	} else {
		test := globalState.FetchTestcase(name)
		if test == nil {
			errorHandler(w, r, http.StatusNotFound)
			return
		}

		if excluded {
			io.WriteString(w, "Excluded OK")
		} else {
			io.WriteString(w, "Unexcluded OK")
		}
		test.SetExcluded(excluded)
	}
}

var globalState *Go262.GlobalState

func Serve(state *Go262.GlobalState) {
	globalState = state

	r := mux.NewRouter()
	r.HandleFunc("/", logReq(indexHandler))
	r.HandleFunc("/suite/{path:.+}", logReq(suiteShowHandler))
	r.HandleFunc("/test/{path:.+}", logReq(testShowHandler))
	r.HandleFunc("/run/{path:.+}", logReq(runTestHandler))
	r.HandleFunc("/read/{runtype}/{path:.+}", logReq(readCodeHandler))
	r.HandleFunc("/logs/{runtype}/{type}/{path:.+}", logReq(readLogsHandler))
	r.HandleFunc("/exclude/{truefalse}/{path:.+}", logReq(setExcludedHandler))

	s := &http.Server{
		Addr:    ":8080",
		Handler: r,
		// ### Can't use this, it will kill long lasting requests
		//ReadTimeout: 10 * time.Second,
		//WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	fmt.Printf("Listening on http://localhost:8080/\n")
	log.Fatal(s.ListenAndServe())
}
