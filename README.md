auditnozzle is a CF pushable app that connects to the firehose endpoint and provides a number of functions for analyzing the flow of messages out of the firehose.

To build and run

`GOOS=linux go build main.go`

`cf push auditnozzle -b binary_buildpack -c "./main"`

To use:
set environment variables `API_ENDPOINT`, `USER_ID`, `USER_PASSWORD` and optionally `SKIP_SSL_VALIDATION` (true/false) 

to see available options: `curl -s auditnozzle.walnut.cf-app.com`

Supported operations (optional paramaters in <>):

- measurelogs

- reportlogs <showguids (default no)>

- measuremetrics

- reportmetricintervals <consolidated (default yes)>

- reportmetrics

- measurelatency

- reportlatency

- measureloghist

- reportloghist

- measuretags

- reporttags <showjobs (default no)

- status

- reset

all scanners take runtime= flag defaults to 1m

example:

`curl -s auditnozzle.walnut.cf-app.com/measurelogs`

Will start a log scanning run.

`curl -s auditnozzle.walnut.cf-app.com/reportlogs`

Will report the results. It can be run either while the scan is still running, or after it is over.

Use `curl -s auditnozzle.walnut.cf-app.com/status` to monitor which scanners are running.



