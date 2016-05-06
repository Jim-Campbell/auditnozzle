package metricparser

import "mauditnozzle/helpers"

type MetricParser struct {
	readMetrics, csvMetrics, UndocumentedMetrics, StaleCsvMetrics []string
}

func NewMetricParser(readMetrics, csvMetrics []string) *MetricParser{
	return &MetricParser{
		readMetrics: readMetrics,
		csvMetrics: csvMetrics,
	}
}

func (p *MetricParser) FindFirehoseMetricsNoDocumentation() {
	var missingMetrics []string
	for _, metric := range p.readMetrics {
		if !helpers.Exists(metric, p.csvMetrics) {
			missingMetrics = append(missingMetrics, metric)
		}
	}
	p.UndocumentedMetrics = missingMetrics
}

func (p *MetricParser) FindCSVMetricsNoFirehose() {
	var missingMetrics []string
	for _, metric := range p.csvMetrics {
		if !helpers.Exists(metric, p.readMetrics) {
			missingMetrics = append(missingMetrics, metric)
		}
	}
	p.StaleCsvMetrics = missingMetrics
}
