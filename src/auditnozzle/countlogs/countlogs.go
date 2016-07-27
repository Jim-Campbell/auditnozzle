package countlogs

import (
	"auditnozzle/scanengine"
	"fmt"
	"github.com/cloudfoundry/sonde-go/events"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

/******************************************************************************************/
type LogType struct {
	guid  string
	name  string
	src   string
	count int
}

type LogMapType map[string]*LogType

type LogSliceType []*LogType

func (a LogSliceType) Len() int      { return len(a) }
func (a LogSliceType) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a LogSliceType) Less(i, j int) bool {
	if a[i].count != a[j].count {
		return a[i].count > a[j].count
	}
	if a[i].name != a[j].name {
		return a[i].name < a[j].name
	}
	return a[i].src < a[j].src
}

type MetricCount struct {
	startValue uint64
	lastValue  uint64
	nowValue   uint64
	count      uint64
	missed     int
	msgs       int
}

/******************************************************************************************/
var (
	CountScan            = scanengine.ScanEngine{Name: "Count Logs"}
	TotalLogsReceived    int
	TotalAppLogsReceived int
	TimeLastSample       time.Time
	CountLastSample      int
	RateLastSecond       int
	DroppedMessages      int

	ReadLogsMap = make(LogMapType)
	MetricMaps  = make(map[string]map[string]*MetricCount)
	LogMutex    sync.Mutex
)

/******************************************************************************************/

func ResetData() {
	CountScan.Reset()
	TotalLogsReceived = 0
	TotalAppLogsReceived = 0
	CountLastSample = 0
	RateLastSecond = 0
	DroppedMessages = 0

	LogMutex.Lock()
	{
		ReadLogsMap = make(LogMapType)
		MetricMaps = make(map[string]map[string]*MetricCount)
	}
	LogMutex.Unlock()

}
func ReadAndCountLogs(req *http.Request, res io.Writer) {

	if err := CountScan.Start(req, res); err != nil {
		return
	}

	// Set up the interval that is used to do rate per second
	TimeLastSample = time.Now()

	go func() {
		CountScan.Run(CountIterator)
	}()

}

func CountIterator(msg *events.Envelope) {

	if msg.GetEventType() == events.Envelope_CounterEvent {
		ProcessCounterMetricMessage(msg)
		return
	}

	if msg.GetEventType() == events.Envelope_LogMessage {
		ProcessLogMessage(msg)
		return
	}
}

/*****************************************************************************************/

func ProcessCounterMetricMessage(msg *events.Envelope) {
	mp := FindCounterMetricMapEntry(msg)
	if mp == nil {
		return
	}

	index := msg.GetIndex()
	total := msg.GetCounterEvent().GetTotal()
	delta := msg.GetCounterEvent().GetDelta()

	LogMutex.Lock()
	defer LogMutex.Unlock()

	fmt.Println("metric", msg.GetCounterEvent().GetName(), total, delta)

	p, ok := mp[index]
	if ok == false {
		m := MetricCount{total, total, total, delta, 0, 1}
		mp[index] = &m
		return
	}

	//detect if we missed a metric message
	if p.lastValue+delta != total {
		p.missed++
	}
	p.msgs++
	p.lastValue = p.nowValue
	p.nowValue = total

}

func FindCounterMetricMapEntry(msg *events.Envelope) map[string]*MetricCount {
	var origin string

	name := msg.GetCounterEvent().GetName()
	origin = msg.GetOrigin()
	tags := msg.GetTags()

	// Diego doesn't yet support tags
	if name == "logSenderTotalMessagesRead" && origin == "rep" {
		return GetMetricMap("Diego")
	}

	// Metron and doppler support tags, so only look at CounterEvents that are tagged as log count metrics
	if tags["event_type"] != "LogMessage" {
		return nil
	}

	if name == "listeners.receivedEnvelopes" && origin == "DopplerServer" {
		return GetMetricMap("Doppler")
	}
	if name == "dropsondeMarshaller.sentEnvelopes" && origin == "MetronAgent" {
		return GetMetricMap("Metron")
	}

	return nil
}

func GetMetricMap(index string) map[string]*MetricCount {

	// *change*
	LogMutex.Lock()
	defer LogMutex.Unlock()

	mc, ok := MetricMaps[index]
	if ok == true {
		return mc
	}

	var me = make(map[string]*MetricCount)
	MetricMaps[index] = me
	return me
}

/*****************************************************************************************/

func ProcessLogMessage(msg *events.Envelope) {

	ProcessLogTiming()
	guid, name, src := FixupLogMessage(msg)
	InsertLogMessageInTable(guid, src, name)
}

func FixupLogMessage(msg *events.Envelope) (string, string, string) {
	var name string

	guid := msg.GetLogMessage().GetAppId()
	src := msg.GetLogMessage().GetSourceType()

	if src == "" {
		src = "==="
	}

	if src == "APP" {
		TotalAppLogsReceived++
	}

	if guid == "system" {
		var cnt int

		n, _ := fmt.Sscanf(string(msg.GetLogMessage().GetMessage()), "Dropped %d message(s) from MetronAgent to Doppler", &cnt)
		if n == 1 {
			name = "system: dropped messages"
			DroppedMessages += cnt
		} else {
			name = "system: unknown message"
		}
	}

	return guid, name, src
}

func ProcessLogTiming() {
	TotalLogsReceived++
	t := time.Now()
	ti := t.Sub(TimeLastSample)

	if ti > time.Second {
		TimeLastSample = t
		lc := TotalLogsReceived - CountLastSample
		RateLastSecond = int(float64(lc) / float64(ti.Seconds()))
		CountLastSample = TotalLogsReceived
	}
}

func InsertLogMessageInTable(guid, src, name string) {

	key := guid + src

	// *change*
	LogMutex.Lock()
	defer LogMutex.Unlock()

	value, ok := ReadLogsMap[key]
	if !ok {
		entry := &LogType{guid, name, src, 1}
		ReadLogsMap[key] = entry
		QueueNameLookup(entry)
		return
	}
	value.count++

}

func SortLogs(logList LogMapType) LogSliceType {
	var sl LogSliceType

	for _, l := range logList {
		sl = append(sl, l)
	}

	sort.Sort(sl)
	return sl
}

/******************************************************************************************/

func ReportCountedLogs(ow io.Writer, showGuid bool) {

	CountScan.WriteStatus(ow)

	LogMutex.Lock()
	mapLen := len(ReadLogsMap)
	LogMutex.Unlock()

	if mapLen == 0 {
		fmt.Fprintln(ow, "No log data collected")
		return
	}

	fmt.Fprintf(ow, "Total logs messages: %8d APP messages: %8d\n", TotalLogsReceived, TotalAppLogsReceived)

	/*************************************************************************************************/
	LogMutex.Lock()

	logList := SortLogs(ReadLogsMap)

	// This is all the data reported by the app name lookup channel logic
	// needs to be in a mutex because the variables all need to be consistent
	lookupCnt := TotalNameLookupCount
	inProcessCnt := NumberOfNamesInProcess
	aveQueueMs := int((TotalNameQueueTime.Seconds() / float64(TotalNameLookupCount)) * 1000)
	maxQueueMs := int(MaxNameQueueTime.Seconds() * 1000)
	minQueueMs := int(MinNameQueueTime.Seconds() * 1000)
	aveLookupMs := int((TotalNameLookupTime.Seconds() / float64(TotalNameLookupCount)) * 1000)
	maxLookupMs := int(MaxNameLookupTime.Seconds() * 1000)
	minLookupMs := int(MinNameLookupTime.Seconds() * 1000)
	maxEnqueue := MaxNumberOfNamesInProcess

	LogMutex.Unlock()
	/*************************************************************************************************/

	PrintLogMetricStats(ow, "Diego")
	PrintLogMetricStats(ow, "Metron")
	PrintLogMetricStats(ow, "Doppler")

	fmt.Fprintln(ow, lookupCnt, "names, ave lookup", aveLookupMs, "max/min", maxLookupMs, minLookupMs, "ave queue", aveQueueMs, "max/min", maxQueueMs, minQueueMs, "queued", inProcessCnt, "max", maxEnqueue)
	fmt.Fprintf(ow, "rate last second %5d total dropped messages %d\n", RateLastSecond, DroppedMessages)

	for _, l := range logList {

		fmt.Fprintf(ow, "%8d %5s %s", l.count, l.src, l.name)

		if showGuid {
			fmt.Fprintf(ow, "| %s", l.guid)
		}
		fmt.Fprintln(ow)
	}

}

func PrintLogMetricStats(ow io.Writer, name string) {

	// *change*
	LogMutex.Lock()
	mm, ok := MetricMaps[name]
	LogMutex.Unlock()

	if ok {
		fmt.Fprintf(ow, "%8s [%2d]:", name, len(mm))
		var count uint64
		var missed int
		var msgs int

		for _, mp := range mm {
			count += mp.count
			missed += mp.missed
			msgs += mp.msgs
		}
		fmt.Fprintf(ow, " %8d [%2d|%8d]", count, missed, msgs)

	}
	fmt.Fprintln(ow)
}

/***************************************************************************************************************************/

type NameLookup struct {
	LogEntry        *LogType
	EnqueueTime     time.Time
	StartLookupTime time.Time
}

var (
	NumberOfNamesInProcess    int
	MaxNumberOfNamesInProcess int
	TotalNameLookupTime       time.Duration
	MinNameLookupTime         time.Duration
	MaxNameLookupTime         time.Duration
	TotalNameQueueTime        time.Duration
	MinNameQueueTime          time.Duration
	MaxNameQueueTime          time.Duration
	TotalNameLookupCount      int
)

var LogNameChannel chan NameLookup = make(chan NameLookup, 20000)

// Send off the name - the manipulations of the globals need to be protected by a mutex. However, the *one* caller of this function
// does it from within a mutex. So we don't project it here. But that is fragile...
func QueueNameLookup(l *LogType) {
	var nl NameLookup

	nl.LogEntry = l
	nl.EnqueueTime = time.Now()

	LogNameChannel <- nl

	// LogMutex.Lock()
	{
		NumberOfNamesInProcess++
		if NumberOfNamesInProcess > MaxNumberOfNamesInProcess {
			MaxNumberOfNamesInProcess = NumberOfNamesInProcess
		}
	}
	// LogMutex.Unlock()
}

func ProcessNameLookup() {

	for {
		nl := <-LogNameChannel

		nl.StartLookupTime = time.Now()

		if nl.LogEntry.name == "" {
			nl.LogEntry.name = CountScan.AppName(nl.LogEntry.guid)
		}

		if nl.LogEntry.name == "" {
			nl.LogEntry.name = "guid: " + nl.LogEntry.guid
		}

		t := time.Now()
		queueTime := t.Sub(nl.EnqueueTime)
		processTime := t.Sub(nl.StartLookupTime)

		LogMutex.Lock()
		{
			TotalNameLookupCount++

			TotalNameQueueTime += queueTime
			if MaxNameQueueTime < queueTime {
				MaxNameQueueTime = queueTime
			}
			if MinNameQueueTime == 0 || MinNameQueueTime > queueTime {
				MinNameQueueTime = queueTime
			}

			TotalNameLookupTime += processTime
			if MaxNameLookupTime < processTime {
				MaxNameLookupTime = processTime
			}
			if MinNameLookupTime == 0 || MinNameLookupTime > processTime {
				MinNameLookupTime = processTime
			}

			NumberOfNamesInProcess--
		}
		LogMutex.Unlock()

	}

}
