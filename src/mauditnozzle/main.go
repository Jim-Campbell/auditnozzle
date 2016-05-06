package main

import (
	"crypto/tls"
	"fmt"
	"os"

	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
	"mauditnozzle/csvreader"
	"time"
	"mauditnozzle/helpers"
	"mauditnozzle/metricparser"
	"flag"
)

var (
	authToken = os.Getenv("CF_ACCESS_TOKEN")  // use $(cf oauth-token | grep bearer)

	metricParser *metricparser.MetricParser

	fileName, firehoseAddr string
	runtime time.Duration
)


const firehoseSubscriptionId = "firehose-a"

func init() {
	flag.StringVar(&fileName, "f", "", "the filename for the documentation CSV metrics")
	flag.DurationVar(&runtime, "t", 10 * time.Second, "The duration to gather metrics from the Firehose; default 10s")
	flag.StringVar(&firehoseAddr, "firehoseAddr", "", "the address of the firehose; format: wss://host:port")
}

func main() {
	flag.Parse()
	validateFlags()

	fmt.Fprintln(os.Stderr, "Reading in metrics from CSV...")
	csvMetrics := getMetricNames(fileName)

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
			fmt.Fprintln(os.Stdout, "You could try resetting the CF_ACCESS_TOKEN env variable")
			os.Exit(-1)
		}
	}()

	return msgChan
}

func readFirehoseMetrics(msgChan <-chan *events.Envelope) []string {
	var readMetrics []string
	timer := time.NewTimer(runtime)
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

func validateFlags() {
	if fileName == "" {
		fmt.Fprintln(os.Stdout, "must provide a file name: -f")
		os.Exit(-1)
	}

	if firehoseAddr == "" {
		fmt.Fprintln(os.Stdout, "must provide an address for the firehose: -firehoseAddr")
		os.Exit(-1)
	}
}

type ConsoleDebugPrinter struct{}

func (c ConsoleDebugPrinter) Print(title, dump string) {
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, dump)
}
