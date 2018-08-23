package mpiostat

import (
	"flag"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

type IostatPlugin struct {
	Prefix string
}

var iostatColumnsPattern = regexp.MustCompile(
	`^(\d+)\s+(\d+)\s+(\S+)\s+(\d+)\s+(\d+?)\s+(\d+?)\s+(\d+?)\s+(\d+?)\s+(\d+?)\s+(\d+?)\s+(\d+?)\s+(\d+?)\s+(\d+?)\s+(\d+?)$`,
)

func (i IostatPlugin) GraphDefinition() map[string]mp.Graphs {
	labelPrefix := strings.Title(i.MetricKeyPrefix())
	return map[string]mp.Graphs{
		"disk.request.#": {
			Label: (labelPrefix + " Requests"),
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "reads", Label: "reads", Diff: true},
				{Name: "writes", Label: "writes", Diff: true},
			},
		},
		"disk.merge.#": {
			Label: (labelPrefix + " Merge"),
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "readsMerged", Label: "reads merged", Diff: true},
				{Name: "writesMerged", Label: "writes merged", Diff: true},
			},
		},
		"disk.traffic.#": {
			Label: (labelPrefix + " Traffic (sectors)"),
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "sectorsRead", Label: "read", Diff: true},
				{Name: "sectorsWritten", Label: "write", Diff: true},
			},
		},
		"disk.time.#": {
			Label: (labelPrefix + " Time (ms)"),
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "readTime", Label: "read", Diff: true},
				{Name: "writeTime", Label: "write", Diff: true},
				{Name: "ioTime", Label: "io", Diff: true},
				{Name: "ioTimeWeighted", Label: "io weighted", Diff: true},
			},
		},
		"disk.inprogress.#": {
			Label: (labelPrefix + " IO in Progress"),
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "ioInProgress", Label: "io"},
			},
		},
	}
}

/*
$ cat /proc/diskstats
 253       0 vda 1535048 279 41601294 520508 73249233 7260487 540931528 10616000 0 5871704 11113052
 253       1 vda1 1534559 279 41576784 520420 46025748 7260487 540931528 8670868 0 3948708 9173652
 253      16 vdb 72583 27934 814612 11784 36796 368511 3242456 23704 0 25272 35452
*/

func (i IostatPlugin) FetchMetrics() (map[string]float64, error) {
	io, err := ioutil.ReadFile("/proc/diskstats")
	if err != nil {
		return nil, fmt.Errorf("Cannot read from file /proc/diskstats: %s", err)
	}
	result := make(map[string]float64)
	for _, line := range strings.Split(string(io), "\n") {
		if matches := strings.Fields(line); len(matches) > 0 {
			// Skip for empty line. See https://github.com/golang/go/issues/13075 for details.
			if len(matches[0]) == 0 {
				continue
			}

			device := matches[2]

			// TODO: Skip virtual devices, such as loop devices.

			// "Discard"s are introduced in Kernel 4.18. See linux/Documentation/iostats.txt for details.
			metricNames := []string{
				"reads", "readsMerged", "sectorsRead", "readTime",
				"writes", "writesMerged", "sectorsWritten", "writeTime",
				"ioInProgress", "ioTime", "ioTimeWeighted",
				"discards", "discardsMerged", "sectorsDiscarded", "discardTime",
			}

			for i, metric := range matches[3:] {
				// TODO: Sanitize these values.
				key := fmt.Sprintf("disk.%s.%s", device, metricNames[i])
				result[key], err = strconv.ParseFloat(metric, 64)
				if err != nil {
					return nil, fmt.Errorf("Failed to parse value: %s", err)
				}
			}
		}
	}

	return result, nil
}

func (i IostatPlugin) MetricKeyPrefix() string {
	if i.Prefix == "" {
		i.Prefix = "iostat"
	}
	return i.Prefix
}

func Do() {
	optPrefix := flag.String("metric-key-prefix", "iostat", "Metric key prefix")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	i := IostatPlugin{
		Prefix: *optPrefix,
	}
	plugin := mp.NewMackerelPlugin(i)
	plugin.Tempfile = *optTempfile
	plugin.Run()
}
