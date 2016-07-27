package counttags

import (
	"auditnozzle/scanengine"
	"fmt"
	"github.com/cloudfoundry/sonde-go/events"
	"io"
	"net/http"
	"sort"
	"sync"
)

/******************************************************************************************/
type TagType struct {
	tagkey   string
	tagvalue string
	origin   string
	job      string
	count    int
}

type TagMapType map[string]*TagType

type TagSliceType []*TagType

func (a TagSliceType) Len() int      { return len(a) }
func (a TagSliceType) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a TagSliceType) Less(i, j int) bool {
	if a[i].job != a[j].job {
		return a[i].job > a[j].job
	}
	if a[i].origin != a[j].origin {
		return a[i].origin > a[j].origin
	}
	if a[i].tagkey != a[j].tagkey {
		return a[i].tagkey < a[j].tagkey
	}
	return a[i].tagvalue < a[j].tagvalue
}

/******************************************************************************************/
var (
	TagsScan          = scanengine.ScanEngine{Name: "Count Tags"}
	TotalMsgsReceived int
	TotalTagsReceived int
	ReadTagsMap       = make(TagMapType)
	TagMutex          sync.Mutex
)

/******************************************************************************************/
func ResetData() {
	TagsScan.Reset()
	TotalMsgsReceived = 0

	TagMutex.Lock()
	{
		ReadTagsMap = make(TagMapType)
	}
	TagMutex.Unlock()

}

func ReadAndCountTags(req *http.Request, res io.Writer) {

	if err := TagsScan.Start(req, res); err != nil {
		return
	}

	go func() {
		TagsScan.Run(TagsIterator)
	}()

}

func TagsIterator(msg *events.Envelope) {

	TotalMsgsReceived++

	tags := msg.GetTags()
	if len(tags) == 0 {
		return
	}

	TotalTagsReceived++
	origin := msg.GetOrigin()
	job := msg.GetJob()

	TagMutex.Lock()
	defer TagMutex.Unlock()

	for k, v := range tags {
		key := origin + job + k + v

		t, ok := ReadTagsMap[key]
		if !ok {
			entry := &TagType{k, v, origin, job, 1}
			ReadTagsMap[key] = entry
			continue
		}
		t.count++
	}

}

/******************************************************************************************/

func ReportCountedTags(ow io.Writer, showJobsFlag bool) {
	var (
		tagList TagSliceType
		outMap  TagMapType
	)

	TagsScan.WriteStatus(ow)

	TagMutex.Lock()
	tmpMap := ReadTagsMap
	TagMutex.Unlock()

	if len(tmpMap) == 0 {
		fmt.Fprintln(ow, "No tag data collected")
		return
	}

	if showJobsFlag {
		outMap = tmpMap
	} else {
		outMap = ConsolidateTags(tmpMap)
	}

	fmt.Fprintf(ow, "Tags map %3d, tagged messages %8d out of %8d messages\n", len(outMap), TotalTagsReceived, TotalMsgsReceived)

	for _, t := range outMap {
		tagList = append(tagList, t)
	}

	sort.Sort(tagList)

	if showJobsFlag {
		fmt.Fprintf(ow, "______________________________")
	}
	fmt.Fprintf(ow, "_____________________________________________________________________\n")

	if showJobsFlag {
		fmt.Fprintf(ow, "               Job              |")
	}
	fmt.Fprintf(ow, "            Origin          |     Key   |        Value     |Count|\n")

	for _, t := range tagList {
		if showJobsFlag {
			fmt.Fprintf(ow, "%-32s|", t.job)
		}
		fmt.Fprintf(ow, "%-28s|%-11s|%-18s|%4d|\n", t.origin, t.tagkey, t.tagvalue, t.count)
	}
}

// take each set of name/index and consolidate
func ConsolidateTags(mapIn TagMapType) TagMapType {

	mapOut := make(TagMapType)

	for _, tIn := range mapIn {

		key := tIn.origin + tIn.tagkey + tIn.tagvalue

		tOut, ok := mapOut[key]
		if !ok {
			tmpTag := &TagType{
				tagkey:   tIn.tagkey,
				tagvalue: tIn.tagvalue,
				origin:   tIn.origin,
				count:    tIn.count,
			}

			mapOut[key] = tmpTag
			continue
		}

		tOut.count += tIn.count

	}
	return mapOut

}
