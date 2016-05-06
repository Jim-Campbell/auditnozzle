package main

import (
	"crypto/tls"
	"fmt"
	"os"

	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/kicombs/mauditnozzle/csvreader"
	"time"
	"github.com/kicombs/mauditnozzle/helpers"
	"github.com/kicombs/mauditnozzle/metricparser"
)

var (
	firehoseAddr = os.Getenv("FIREHOSE_ADDR") // should look like ws://host:port
	authToken = os.Getenv("CF_ACCESS_TOKEN")  // use $(cf oauth-token | grep bearer)
	metricParser *metricparser.MetricParser
)


const firehoseSubscriptionId = "firehose-a"

func main() {
	fmt.Fprintln(os.Stderr, "Reading in metrics from CSV...")
	csvMetrics := getMetricNames("metrics.list.example.csv")

	fmt.Fprintln(os.Stderr, "Starting the firehose...")
	msgChan := startFirehose()

	fmt.Fprintln(os.Stderr, "Reading In Metrics...")
	readMetrics := readFirehoseMetrics(msgChan)
	fmt.Fprintln(os.Stderr, "Finished reading from the firehose.")

	metricParser = metricparser.NewMetricParser(readMetrics, csvMetrics)

	metricParser.FindFirehoseMetricsNoDocumentation()
	metricParser.FindCSVMetricsNoFirehose()
	fmt.Fprintf(os.Stdout, "Read: %d, CSV: %d, MissingFirehose: %d, MissingCsvMetrics: %d\n", len(readMetrics), len(csvMetrics), len(metricParser.UndocumentedMetrics), len(metricParser.StaleCsvMetrics))

	fmt.Fprintf(os.Stdout, "\n\nUndocumented Firehose Metrics:\n")
	for _, metric := range metricParser.UndocumentedMetrics {
		fmt.Fprintln(os.Stdout, metric)
	}

	fmt.Fprintf(os.Stdout, "\n\nUnused Documented Metrics:\n")
	for _, metric := range metricParser.StaleCsvMetrics {
		fmt.Fprintln(os.Stdout, metric)
	}
}

func getMetricNames(filename string) []string {
	reader := csvreader.NewCsvReader(filename)
	return reader.Metrics
}

func startFirehose() <-chan *events.Envelope {
	connection := consumer.New(firehoseAddr, &tls.Config{InsecureSkipVerify: true}, nil)
	connection.SetDebugPrinter(ConsoleDebugPrinter{})

	fmt.Fprintln(os.Stderr, "===== Streaming Firehose (will only succeed if you have admin credentials)")

	msgChan, errorChan := connection.Firehose(firehoseSubscriptionId, authToken)
	go func() {
		for err := range errorChan {
			fmt.Fprintf(os.Stdout, "%v\n", err.Error())
			os.Exit(-1)
		}
	}()

	return msgChan
}

func readFirehoseMetrics(msgChan <-chan *events.Envelope) []string {
	var readMetrics []string
	timer := time.NewTimer(5 * time.Second)
	for {
		select {
		case <-timer.C:
			return readMetrics
		default:
			msg := <-msgChan
			name := parseMetricName(msg)
			if !helpers.Exists(name, readMetrics) && name != "" {
				readMetrics = append(readMetrics, name)
			}
		}
	}
	return []string{}
}

func parseMetricName(msg *events.Envelope) string {
	switch msg.GetEventType() {
	case events.Envelope_ValueMetric:
		return msg.GetValueMetric().GetName()
	case events.Envelope_CounterEvent:
		return msg.GetCounterEvent().GetName()
	}
	return ""
}

type ConsoleDebugPrinter struct{}

func (c ConsoleDebugPrinter) Print(title, dump string) {
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, dump)
}
