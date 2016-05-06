package main

import (
	"crypto/tls"
	"fmt"
	"os"

	"github.com/cloudfoundry/noaa/consumer"
	"github.com/kicombs/mauditnozzle/csvreader"
)

// FUTURE if time, removing metrics after TTL?

var firehoseAddr = os.Getenv("FIREHOSE_ADDR") // should look like ws://host:port
var authToken = os.Getenv("CF_ACCESS_TOKEN")   // use $(cf oauth-token | grep bearer)

const firehoseSubscriptionId = "firehose-a"

func main() {
	fmt.Println("Reading in metrics from CSV...")
	csvMetrics := getMetricNames("example.csv")
	_ = csvMetrics

	fmt.Println("Starting the firehose...")
	startFirehose()
}

func getMetricNames(filename string) []string{
	reader := csvreader.NewCsvReader(filename)
	return reader.Metrics
}

func startFirehose() {
	connection := consumer.New(firehoseAddr, &tls.Config{InsecureSkipVerify: true}, nil)
	connection.SetDebugPrinter(ConsoleDebugPrinter{})

	fmt.Println("===== Streaming Firehose (will only succeed if you have admin credentials)")

	msgChan, errorChan := connection.Firehose(firehoseSubscriptionId, authToken)
	go func() {
		for err := range errorChan {
			fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		}
	}()

	for msg := range msgChan {
		fmt.Printf("%v \n", msg)
	}
}

type ConsoleDebugPrinter struct{}

func (c ConsoleDebugPrinter) Print(title, dump string) {
	println(title)
	println(dump)
}