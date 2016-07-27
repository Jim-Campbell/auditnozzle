package loglength

import (
	"auditnozzle/helpers"
	"auditnozzle/scanengine"
	"github.com/cloudfoundry/sonde-go/events"
	"io"
	"net/http"
)

var LogLengthHistBin *helpers.HistogramBin = helpers.NewBin(200, 10000)
var LogLengthHistScan = scanengine.ScanEngine{Name: "Log Length Histogram"}

func ResetData() {
	LogLengthHistBin = helpers.NewBin(200, 10000)
	LogLengthHistScan.Reset()
}

func ReadLogHistogram(req *http.Request, res io.Writer) {

	if err := LogLengthHistScan.Start(req, res); err != nil {
		return
	}

	go func() {
		LogLengthHistScan.Run(LogHistIterator)
	}()
}

func LogHistIterator(msg *events.Envelope) {

	if msg.GetEventType() != events.Envelope_LogMessage {
		return
	}

	length := len(msg.GetLogMessage().GetMessage())
	LogLengthHistBin.InsertSample(length)

}

func ReportLogHistogram(outputWriter io.Writer) {
	LogLengthHistScan.WriteStatus(outputWriter)
	LogLengthHistBin.PrintBins(outputWriter)
}
