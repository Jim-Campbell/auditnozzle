package csvreader

import (
	"encoding/csv"
	"io"
	"io/ioutil"
	"log"
	"strings"
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

		if !exists(record[0], metrics) {
			metrics = append(metrics, record[0])
		}
	}
	c.Metrics = metrics
}

func exists(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
