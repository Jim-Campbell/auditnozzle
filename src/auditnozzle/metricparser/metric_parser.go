package metricparser

import (
	"auditnozzle/scanengine"
	"encoding/csv"
	"fmt"
	"github.com/cloudfoundry/sonde-go/events"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

/******************************************************************************************/

type aMetric struct {
	Name                string
	Origin              string
	Job                 string
	Index               string
	IP                  string
	LastTimeReceived    time.Time
	NumberReceived      int
	SumOfAllTimes       time.Duration
	LongestTimeBetween  time.Duration
	ShortestTimeBetween time.Duration
}

type metricSlice []*aMetric
type metricMap map[string]*aMetric

func (a metricSlice) Len() int           { return len(a) }
func (a metricSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a metricSlice) Less(i, j int) bool { return a[i].Origin+a[i].Name < a[j].Origin+a[j].Name }

/******************************************************************************************/

var (
	AuditScan      = scanengine.ScanEngine{Name: "Metric Audit"}
	ReadMetricsMap metricMap
	MetricsMutex   sync.Mutex
)

func ResetData() {
	AuditScan.Reset()

	// *change*
	MetricsMutex.Lock()
	{
		ReadMetricsMap = make(metricMap)
	}
	MetricsMutex.Unlock()
}

func AuditMetrics(req *http.Request, res io.Writer) {

	//Unlike the other monitors, this one can't run adding to history because of the time of the last emitted metric
	//will be from the previous run. So we need to zero it each time
	ResetData()

	if err := AuditScan.Start(req, res); err != nil {
		return
	}

	go func() {
		AuditScan.Run(AuditIterator)
	}()

}

func AuditIterator(msg *events.Envelope) {

	if msg.GetEventType() != events.Envelope_ValueMetric && msg.GetEventType() != events.Envelope_CounterEvent {
		return
	}

	timeNow := time.Now()
	name := ParseMetricName(msg)
	key := msg.GetIndex() + name

	// When doing the Unlock() as a defer, is there a convention for enclosing the protected code in {}s?
	MetricsMutex.Lock()
	defer MetricsMutex.Unlock()

	metric, ok := ReadMetricsMap[key]

	if !ok {

		tmpMetric := aMetric{
			Name:             name,
			Origin:           msg.GetOrigin(),
			Job:              msg.GetJob(),
			Index:            msg.GetIndex(),
			IP:               msg.GetIp(),
			LastTimeReceived: timeNow,
			NumberReceived:   1,
		}
		ReadMetricsMap[key] = &tmpMetric
		return
	}

	metric.UpdateTimeGap(timeNow)

}

func ParseMetricName(msg *events.Envelope) string {

	switch msg.GetEventType() {

	case events.Envelope_ValueMetric:
		return msg.GetValueMetric().GetName()

	case events.Envelope_CounterEvent:
		return msg.GetCounterEvent().GetName()

	default:
		return "unknown_metric_type"
	}
}

func NumberOfMetrics() int {
	MetricsMutex.Lock()
	mapLen := len(ReadMetricsMap)
	MetricsMutex.Unlock()
	return mapLen
}

/******************************************************************************************/

func ReportMetricIntervals(w io.Writer, consolidatedFlag bool) {
	var printMetrics metricMap

	AuditScan.WriteStatus(w)

	if NumberOfMetrics() == 0 {
		fmt.Fprintln(w, "No metric data collected")
		return
	}

	MetricsMutex.Lock()
	{
		if consolidatedFlag {
			printMetrics = ConsolidateMetrics(ReadMetricsMap)
		} else {
			printMetrics = ReadMetricsMap
		}
	}
	MetricsMutex.Unlock()

	PrintMetricTable(printMetrics, w)
}

// Job and Index only value in some cases. Print them only if Index is present
func PrintMetricTable(metricList metricMap, w io.Writer) {

	fmt.Fprintf(w, "-Have recorded %d metrics\n", NumberOfMetrics())
	fmt.Fprintln(w, "_____________________________________________________________________________________________________________")
	fmt.Fprintln(w, "	          Source      |                 Name                               |   num  | ave | max | min |")

	ms := sortMetrics(metricList)

	for _, aMetric := range ms {
		fmt.Fprintf(w, "%-28s|", aMetric.Origin)

		if aMetric.Index != "" {
			fmt.Fprintf(w, "%-32s|", aMetric.Job)
		}

		fmt.Fprintf(w, "%-52s|", aMetric.Name)
		fmt.Fprintf(w, "%8d|", aMetric.NumberReceived)

		if aMetric.NumberReceived > 1 {
			// the '- 1' is because we have n-1 "intervals" recorded when we captured n messages
			averageTime := aMetric.SumOfAllTimes.Seconds() / float64(aMetric.NumberReceived-1)

			fmt.Fprintf(w, "%5.1f|", averageTime)

		} else {
			fmt.Fprintf(w, "   --|")
		}

		fmt.Fprintf(w, "%5.1f|", aMetric.LongestTimeBetween.Seconds())
		fmt.Fprintf(w, "%5.1f|", aMetric.ShortestTimeBetween.Seconds())

		if aMetric.Index != "" {
			fmt.Fprintf(w, "   %s", aMetric.IP)
		}

		fmt.Fprintln(w)

	}

}

// take each set of name/index and consolidate
func ConsolidateMetrics(mapIn metricMap) metricMap {

	mapOut := make(metricMap)

	for _, theMetric := range mapIn {

		key := theMetric.Origin + theMetric.Name

		metric, ok := mapOut[key]
		if !ok {
			tmpMetric := &aMetric{
				Name:                theMetric.Name,
				Origin:              theMetric.Origin,
				Index:               "",
				NumberReceived:      theMetric.NumberReceived,
				SumOfAllTimes:       theMetric.SumOfAllTimes,
				ShortestTimeBetween: theMetric.ShortestTimeBetween,
				LongestTimeBetween:  theMetric.LongestTimeBetween,
			}

			mapOut[key] = tmpMetric
			continue
		}

		metric.Consolidate(theMetric)

	}
	return mapOut

}

func (m *aMetric) Consolidate(inMetric *aMetric) {

	m.NumberReceived += inMetric.NumberReceived - 1
	m.SumOfAllTimes += inMetric.SumOfAllTimes

	if m.ShortestTimeBetween > inMetric.ShortestTimeBetween {
		m.ShortestTimeBetween = inMetric.ShortestTimeBetween
	}
	if m.LongestTimeBetween < inMetric.LongestTimeBetween {
		m.LongestTimeBetween = inMetric.LongestTimeBetween
	}

}

func sortMetrics(mm metricMap) metricSlice {
	var s metricSlice

	for _, aMetric := range mm {
		s = append(s, aMetric)

	}
	sort.Sort(s)
	return s
}

func (m *aMetric) UpdateTimeGap(t time.Time) {

	timeGap := t.Sub(m.LastTimeReceived)

	if m.LongestTimeBetween < timeGap {
		m.LongestTimeBetween = timeGap
	}
	if m.ShortestTimeBetween == 0 || m.ShortestTimeBetween > timeGap {
		m.ShortestTimeBetween = timeGap
	}

	m.SumOfAllTimes += timeGap
	m.LastTimeReceived = t
	m.NumberReceived++
}

/******************************************************************************************/

func ReportMetricDocs(w io.Writer) {

	AuditScan.WriteStatus(w)

	if NumberOfMetrics() == 0 {
		fmt.Fprintln(w, "No metric data collected")
		return
	}

	CSVMetrics, err := ReadCSVMetrics("/app/resources/metrics.list.example.csv")
	if err != nil {
		fmt.Fprintln(w, err.Error())
		return
	}

	MetricsMutex.Lock()
	metrics := ConsolidateMetrics(ReadMetricsMap)
	MetricsMutex.Unlock()

	fmt.Fprintf(w, "\n\n===============> Read: %d metrics from firehose, %d metrics from CSV\n", len(metrics), len(CSVMetrics))

	PrintMetrics(w, "Documented Firehose Metrics:\n", "<", FindFirehoseMetricsInDocumentation(metrics, CSVMetrics))
	PrintMetrics(w, "Undocumented Firehose Metrics:\n", ">", FindFirehoseMetricsNotInDocumentation(metrics, CSVMetrics))
	//	PrintMetrics(w, "Emitted Documented Metrics:\n", "*", FindCSVMetricsInFirehose(metrics, CSVMetrics))
	PrintMetrics(w, "Unemitted Documented Metrics:\n", "*", FindCSVMetricsNotInFirehose(metrics, CSVMetrics))

}

func ReadCSVMetrics(csvFilename string) (metricMap, error) {

	fileData, err := ioutil.ReadFile(csvFilename)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(strings.NewReader(string(fileData)))

	fmt.Fprintln(os.Stdout, "opened reader")
	CsvMetrics := make(metricMap)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		_, ok := CsvMetrics[record[0]]
		if !ok {
			m := &aMetric{
				Name:   record[1],
				Origin: record[0],
			}
			CsvMetrics[m.Origin+m.Name] = m
		}

	}
	return CsvMetrics, nil
}

func FindFirehoseMetricsInDocumentation(metrics metricMap, CsvMetrics metricMap) metricSlice {
	var foundMetrics metricSlice

	for _, m := range metrics {
		if _, ok := CsvMetrics[m.Origin+m.Name]; ok {
			n := &aMetric{
				Name:   m.Name,
				Origin: m.Origin,
			}
			foundMetrics = append(foundMetrics, n)
		}
	}
	return foundMetrics
}

func FindFirehoseMetricsNotInDocumentation(metrics metricMap, CsvMetrics metricMap) metricSlice {
	var missingMetrics metricSlice

	for _, m := range metrics {
		if _, ok := CsvMetrics[m.Origin+m.Name]; !ok {
			n := &aMetric{
				Name:   m.Name,
				Origin: m.Origin,
			}
			missingMetrics = append(missingMetrics, n)
		}
	}
	return missingMetrics
}

func FindCSVMetricsNotInFirehose(metrics metricMap, CsvMetrics metricMap) metricSlice {
	var missingMetrics metricSlice

	for k, cm := range CsvMetrics {
		if _, ok := metrics[k]; !ok {
			m := &aMetric{
				Name:   cm.Name,
				Origin: cm.Origin,
			}
			missingMetrics = append(missingMetrics, m)
		}
	}
	return missingMetrics
}

func FindCSVMetricsInFirehose(metrics metricMap, CsvMetrics metricMap) metricSlice {
	var missingMetrics metricSlice

	for k, cm := range CsvMetrics {
		if _, ok := metrics[k]; ok {
			m := &aMetric{
				Name:   cm.Name,
				Origin: cm.Origin,
			}
			missingMetrics = append(missingMetrics, m)
		}
	}
	return missingMetrics
}

func PrintMetrics(w io.Writer, label string, prefix string, metricList metricSlice) {
	sort.Sort(metricList)
	fmt.Fprintf(w, "\n\n===============> %d %s", len(metricList), label)
	for _, metric := range metricList {
		fmt.Fprintf(w, "%s %-28s| %s\n", prefix, metric.Origin, metric.Name)
	}

}
