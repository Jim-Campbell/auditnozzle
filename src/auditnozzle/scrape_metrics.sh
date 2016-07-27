#!/usr/bin/env bash
echo "Collecting new list of metrics from docs-loggregator..."

awk -f awk_scrape.awk metrics/_cloud_controller.html.md.erb > resources/metrics.list.example.csv
awk -f awk_scrape.awk metrics/_dea_logging_agent.html.md.erb >> resources/metrics.list.example.csv
awk -f awk_scrape.awk metrics/_diego.html.md.erb >> resources/metrics.list.example.csv
awk -f awk_scrape.awk metrics/_doppler.html.md.erb >> resources/metrics.list.example.csv
awk -f awk_scrape.awk metrics/_etcd.html.md.erb >> resources/metrics.list.example.csv
awk -f awk_scrape.awk metrics/_hm9000.html.md.erb >> resources/metrics.list.example.csv
awk -f awk_scrape.awk metrics/_metron_agent.html.md.erb >> resources/metrics.list.example.csv
awk -f awk_scrape.awk metrics/_routing.html.md.erb >> resources/metrics.list.example.csv
awk -f awk_scrape.awk metrics/_syslog_drain_binder.html.md.erb >> resources/metrics.list.example.csv
awk -f awk_scrape.awk metrics/_traffic_controller.html.md.erb >> resources/metrics.list.example.csv
awk -f awk_scrape.awk metrics/_uaa.html.md.erb >> resources/metrics.list.example.csv


echo "List compiled in resources/metrics.list.example.csv"