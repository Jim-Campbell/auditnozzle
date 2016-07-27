package scanengine

import (
	"auditnozzle/firehose"
	"auditnozzle/helpers"
	"errors"
	"fmt"
	"github.com/cloudfoundry/sonde-go/events"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

/********************************************************************************************************
* prototyping a general solution of a scan engine object that does the looping
 */

type ScanEngine struct {
	Name           string
	runtime        time.Duration
	running        bool
	TotalRuntime   time.Duration
	StartTime      time.Time
	CurrentRuntime time.Duration
	RuntimeSoFar   time.Duration
	firehose       <-chan *events.Envelope
}

type Scanner interface {
	Start(req *http.Request, res io.Writer)
	Stop()
	UpdateTime()
	Run(iterator func(*events.Envelope))
	WriteStatus(ow io.Writer)
}

/******************************************************************************************/

func GetRuntime(req *http.Request) time.Duration {

	runtime, err := time.ParseDuration(req.FormValue("runtime"))
	if runtime == 0 || err != nil {
		runtime = 10 * time.Minute
	}
	return runtime
}

func (s *ScanEngine) Reset() {
	s.runtime = 0
	s.running = false
	s.TotalRuntime = 0
	s.CurrentRuntime = 0
	s.RuntimeSoFar = 0

}

func (s *ScanEngine) Start(req *http.Request, res io.Writer) error {
	var err error

	if s.running {
		fmt.Fprintf(res, "%s scanner already running, %s into a run of %s. ", s.Name, helpers.TimeStr(s.RuntimeSoFar), helpers.TimeStr(s.CurrentRuntime))
		return errors.New("scanner already running")
	}

	s.running = true
	s.runtime = GetRuntime(req)

	fmt.Fprintf(os.Stdout, "Starting %s: runtime %s\n", s.Name, s.runtime.String())
	fmt.Fprintf(res, "%s: runtime %s\n", s.Name, s.runtime.String())

	s.CurrentRuntime = s.runtime
	s.StartTime = time.Now()
	s.RuntimeSoFar = 0

	s.firehose, err = firehose.OpenFirehose("auditnozzle-" + strings.Replace(s.Name, " ", "-", -1))
	if err != nil {
		fmt.Fprintln(res, err)
		fmt.Fprintln(os.Stderr, err)
		s.running = false
		return err
	}

	return nil
}

func (s *ScanEngine) Stop() {
	s.TotalRuntime += s.CurrentRuntime
	s.CurrentRuntime = 0
	s.RuntimeSoFar = 0
	s.running = false
}

func (s *ScanEngine) UpdateTime() {
	s.RuntimeSoFar = time.Now().Sub(s.StartTime)

}

func (s *ScanEngine) Run(iterator func(*events.Envelope)) {

	fmt.Fprintf(os.Stdout, "Started acquiring data for %s with timer %s\n", s.Name, s.runtime.String())

	s.RunIterator(iterator)

	s.Stop()

	fmt.Fprintf(os.Stdout, "Stopped %s\n", s.Name)

}

func (s *ScanEngine) RunIterator(iterator func(*events.Envelope)) {

	timer := time.NewTimer(s.runtime)
	for {
		select {
		case <-timer.C:
			return

		default:
			msg := <-s.firehose
			s.UpdateTime()

			iterator(msg)
		}
	}
}

func (s *ScanEngine) WriteStatus(ow io.Writer) {
	if s.running {
		fmt.Fprintf(ow, "%-20s %s|%s ", s.Name, helpers.TimeStr(s.RuntimeSoFar), helpers.TimeStr(s.CurrentRuntime))
	} else {
		fmt.Fprintf(ow, "%-20s -- -- --|-- -- -- ", s.Name)
	}

	fmt.Fprintf(ow, "(%s)\n", helpers.TimeStr(s.TotalRuntime+s.RuntimeSoFar))

}

func (s *ScanEngine) AppName(guid string) string {

	name, err := firehose.AppName(guid)
	if err != nil {
		return fmt.Sprintf("error on name lookup %v\n", err)
	}
	return name
}
