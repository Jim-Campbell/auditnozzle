package latency

import (
	"auditnozzle/helpers"
	"auditnozzle/scanengine"
	"github.com/cloudfoundry/sonde-go/events"
	"io"
	"net/http"
	"time"
)

var MsgLatencyBin *helpers.HistogramBin = helpers.NewBin(20, 200)
var MsgLatencyScan = scanengine.ScanEngine{Name: "Envelope Latency"}

func ResetData() {
	MsgLatencyScan.Reset()
	MsgLatencyBin = helpers.NewBin(20, 200)
}

func MeasureLatency(req *http.Request, res io.Writer) {

	if err := MsgLatencyScan.Start(req, res); err != nil {
		return
	}

	go func() {
		MsgLatencyScan.Run(LatencyIterator)
	}()

}

func LatencyIterator(msg *events.Envelope) {

	now := time.Now().UnixNano()
	timeSent := msg.GetTimestamp()

	latency := now - timeSent
	latencyMs := int(latency / 1e6)

	MsgLatencyBin.InsertSample(latencyMs)

}

func ReportLatency(outputWriter io.Writer) {
	MsgLatencyScan.WriteStatus(outputWriter)
	MsgLatencyBin.PrintBins(outputWriter)

}
