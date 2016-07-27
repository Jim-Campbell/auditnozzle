/**********

bugs

questions

- What happpens when I open the firehose, don't close it, and open again? Sharded? Can I close it?


todos

- put locks around the maps I read and write aysnc. How important is this really?
https://blog.golang.org/go-maps-in-action,
https://golang.org/pkg/sync/#RWMutex

- give a pattern and only count logs for app names with that pattern

- monitor timestamp format in all the log messages

- add tag monitoring to metric audit - not exactly sure what to look for - for now just capture a list of used tags and maybe which component emits them

- check and tune behavior if the nozzle can't keep up - what slow consumer messages from TC and droppled log messages from Doppler should it look for? Simulate by putting delays in the read loop.

- track latency of messages based on the component they are coming from - ave, min, max

- consider tracking intervals of container metrics, explicitly ignored at the moment.

- compare intervals between metrics to docs, once interval is in docs

*/

package main

import (
	"auditnozzle/countlogs"
	"auditnozzle/counttags"
	"auditnozzle/latency"
	"auditnozzle/loglength"
	"auditnozzle/metricparser"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

var (
	ApiEndpoint       string
	CSVfilename       string
	metricsFilename   string
	Password          string
	SkipSSLValidation bool
	UserName          string
	OutputWriter      io.Writer = os.Stdout
)

func main() {

	var err error

	fmt.Fprintln(os.Stdout, "Starting auditnozzle\n")

	http.HandleFunc("/measurelogs", countLogsResponse)
	http.HandleFunc("/reportlogs", reportLogsResponse)
	http.HandleFunc("/measuremetrics", auditMetricsResponse)
	http.HandleFunc("/reportmetricintervals", reportMetricIntervalssResponse)
	http.HandleFunc("/reportmetricdocs", reportMetricDocsResponse)
	http.HandleFunc("/measurelatency", measureLatencyResponse)
	http.HandleFunc("/reportlatency", reportLatencyResponse)
	http.HandleFunc("/measureloghist", measureLogHistogramResponse)
	http.HandleFunc("/reportloghist", reportLogHistogramResponse)
	http.HandleFunc("/measuretags", measureTagsResponse)
	http.HandleFunc("/reporttags", reportTagsResponse)
	http.HandleFunc("/status", statusResponse)
	http.HandleFunc("/reset", resetResponse)
	http.HandleFunc("/", defaultResponse)

	go countlogs.ProcessNameLookup()

	fmt.Println("listening for auditnozzle commands via curl")
	err = http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		panic(err)
	}

}

// ???? Why does this satisfy the interface w/o Handle type somehow involved
func defaultResponse(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(res, "Supported operations:")
	fmt.Fprintln(res, "curl <host URL>/<operation>?<parm>=<value>")
	fmt.Fprintln(res, " measurelogs")
	fmt.Fprintln(res, " reportlogs <showguids (default no)>")
	fmt.Fprintln(res, " measuremetrics")
	fmt.Fprintln(res, " reportmetricintervals <consolidated (default yes)>")
	fmt.Fprintln(res, " reportmetrics")
	fmt.Fprintln(res, " measurelatency")
	fmt.Fprintln(res, " reportlatency")
	fmt.Fprintln(res, " measureloghist")
	fmt.Fprintln(res, " reportloghist")
	fmt.Fprintln(res, " measuretags")
	fmt.Fprintln(res, " reporttags <showjobs (default no)")
	fmt.Fprintln(res, " status")
	fmt.Fprintln(res, " reset")

	fmt.Fprintln(res, "-- all scanners take runtime= flag defaults to 1m")
	fmt.Fprintln(res, "Set ENV variables: API_ENDPOINT, USER_ID, USER_PASSWORD")
	fmt.Fprintln(res, "Optionally set SKIP_SSL_VALIDATION")

}

//curl "auditnozzle.walnut.cf-app.com/countlogs?runtime=20s”
func countLogsResponse(res http.ResponseWriter, req *http.Request) {
	countlogs.ReadAndCountLogs(req, res)
}

func reportLogsResponse(res http.ResponseWriter, req *http.Request) {
	countlogs.ReportCountedLogs(res, GetGuidFlag(req))
}

func auditMetricsResponse(res http.ResponseWriter, req *http.Request) {
	metricparser.AuditMetrics(req, res)
}

func reportMetricIntervalssResponse(res http.ResponseWriter, req *http.Request) {
	metricparser.ReportMetricIntervals(res, GetConsolidatedFlag(req))
}

func reportMetricDocsResponse(res http.ResponseWriter, req *http.Request) {
	metricparser.ReportMetricDocs(res)
}

func measureLatencyResponse(res http.ResponseWriter, req *http.Request) {
	latency.MeasureLatency(req, res)
}

func reportLatencyResponse(res http.ResponseWriter, req *http.Request) {
	latency.ReportLatency(res)
}

func measureLogHistogramResponse(res http.ResponseWriter, req *http.Request) {
	loglength.ReadLogHistogram(req, res)
}

func reportLogHistogramResponse(res http.ResponseWriter, req *http.Request) {
	loglength.ReportLogHistogram(res)
}

func measureTagsResponse(res http.ResponseWriter, req *http.Request) {
	counttags.ReadAndCountTags(req, res)
}

func reportTagsResponse(res http.ResponseWriter, req *http.Request) {
	counttags.ReportCountedTags(res, GetShowJobsFlag(req))
}

func statusResponse(res http.ResponseWriter, req *http.Request) {
	countlogs.CountScan.WriteStatus(res)
	loglength.LogLengthHistScan.WriteStatus(res)
	latency.MsgLatencyScan.WriteStatus(res)
	metricparser.AuditScan.WriteStatus(res)
}

func resetResponse(res http.ResponseWriter, req *http.Request) {
	countlogs.ResetData()
	loglength.ResetData()
	latency.ResetData()
	metricparser.ResetData()
}

func GetGuidFlag(req *http.Request) bool {
	showGuids, err := strconv.ParseBool(req.FormValue("showguid"))
	if err != nil {
		showGuids = false
	}
	return showGuids
}

func GetConsolidatedFlag(req *http.Request) bool {

	consolidatedFlag, err := strconv.ParseBool(req.FormValue("consolidated"))
	if err != nil {
		consolidatedFlag = true
	}
	return consolidatedFlag
}

func GetShowJobsFlag(req *http.Request) bool {

	showjobsFlag, err := strconv.ParseBool(req.FormValue("showjobs"))
	if err != nil {
		showjobsFlag = false
	}
	return showjobsFlag
}
