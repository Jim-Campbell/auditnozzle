package helpers

import (
	"fmt"
	"io"
	"time"
)

//****************************************************************************************
// General histogram bin

type HistogramBin struct {
	bins      []int
	increment int
	totalCnt  int
	totalVal  int
	max       int
	numGTmax  int
	lowest    int
	highest   int
}

func NewBin(increment, max int) *HistogramBin {

	numBins := (max / increment) + 1

	return &HistogramBin{
		increment: increment,
		max:       max,
		bins:      make([]int, numBins), // needs to be numBins size
	}
}

func (h *HistogramBin) InsertSample(s int) {

	// fmt.Println("insert", s, h.max, s/h.increment, len(h.bins))

	if s > h.highest {
		h.highest = s
	}

	if h.lowest == 0 || s < h.lowest {
		h.lowest = s
	}

	h.totalCnt++
	h.totalVal += s

	if s >= h.max {
		h.numGTmax++
		return
	}

	i := s / h.increment

	h.bins[i]++

}

func (h *HistogramBin) PrintBins(iow io.Writer) {

	if h.totalCnt == 0 {
		fmt.Fprintln(iow, "No histogram data recorded")
		return
	}

	fmt.Fprintln(iow, "________________________________________")
	fmt.Fprintln(iow, "|  low   |  high  |  count  | percent |")
	fmt.Fprintln(iow, "________________________________________")

	for i := 0; i < len(h.bins); i++ {
		fmt.Fprintf(iow, "|%7d |%7d |%8d |%8d |\n", i*h.increment, (i+1)*h.increment, h.bins[i], 100*h.bins[i]/h.totalCnt)
	}
	fmt.Fprintf(iow, "Num greater than max of %d: %d (%d)\n", h.max, h.numGTmax, 100*h.numGTmax/h.totalCnt)

	fmt.Fprintf(iow, "Max %d\n", h.highest)
	fmt.Fprintf(iow, "Min %d\n", h.lowest)
	fmt.Fprintf(iow, "Average %d\n", h.totalVal/h.totalCnt)
	fmt.Fprintf(iow, "N %d\n", h.totalCnt)

}

/******************************************************************************************/
// general helpers

func TimeStr(t time.Duration) string {

	secs := int(t.Seconds()) % 60
	mins := int(t.Minutes()) % 60
	hours := int(t.Hours())
	return fmt.Sprintf("%02d:%02d:%02d", hours, mins, secs)

}

func Exists(a string, list map[string]string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
