package csvreader

import (
	"encoding/csv"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"mauditnozzle/helpers"
)

type csvReader struct{
	Metrics []string
}

func NewCsvReader(filename string) *csvReader {
	reader := csvReader{}
	reader.readFromFile(filename)
	return &reader
}

func (c *csvReader) readFromFile(filename string) {
	fileData, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	reader := csv.NewReader(strings.NewReader(string(fileData)))

	var metrics []string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if !helpers.Exists(record[0], metrics) {
			metrics = append(metrics, record[0])
		}
	}
	c.Metrics = metrics
}