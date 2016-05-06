package metricparser

import (
	"mauditnozzle/helpers"
	"fmt"
)

type MetricParser struct {
	readMetrics map[string]string
	csvMetrics, UndocumentedMetrics, StaleCsvMetrics []string
}

func NewMetricParser(readMetrics map[string]string, csvMetrics []string) *MetricParser{
	return &MetricParser{
		readMetrics: readMetrics,
		csvMetrics: csvMetrics,
	}
}

func (p *MetricParser) FindFirehoseMetricsNoDocumentation() {
	var missingMetrics []string
	for metric, origin := range p.readMetrics {
		if !helpers.Exists(metric, p.csvMetrics) {
			missingMetrics = append(missingMetrics, fmt.Sprintf("%s=%s", origin, metric))
		}
	}
	p.UndocumentedMetrics = missingMetrics
}

func (p *MetricParser) FindCSVMetricsNoFirehose() {
	var missingMetrics []string
	for _, metric := range p.csvMetrics {
		if _, ok := p.readMetrics[metric]; !ok {
			missingMetrics = append(missingMetrics, metric)
		}
	}
	p.StaleCsvMetrics = missingMetrics
}
