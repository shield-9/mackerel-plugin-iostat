package mpiostat

import (
	"encoding/json"
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
		"device.request.#": {
			Label: (labelPrefix + " Device Utilization - Requests"),
			Unit:  mp.UnitIOPS,
			Metrics: []mp.Metrics{
				{Name: "read_merged", Label: "read merged"},
				{Name: "write_merged", Label: "write merged"},
				{Name: "read_completed", Label: "read completed"},
				{Name: "write_completed", Label: "write completed"},
			},
		},
		"device.transfer.#": {
			Label: (labelPrefix + "  Device Utilization - Transfer"),
			Unit:  mp.UnitBytesPerSecond,
			Metrics: []mp.Metrics{
				{Name: "read", Label: "read", Scale: 1024, Stacked: true},
				{Name: "write", Label: "write", Scale: 1024, Stacked: true},
			},
		},
		"device.await.#": {
			Label: (labelPrefix + " Device Utilization - Await"),
			Unit:  mp.UnitFloat,
			Metrics: []mp.Metrics{
				{Name: "total", Label: "total"},
				{Name: "read", Label: "read", Stacked: true},
				{Name: "write", Label: "write", Stacked: true},
				{Name: "svctm", Label: "svctm"},
			},
		},
		"device.percentage.#": {
			Label: (labelPrefix + " Device Utilization - Percentage"),
			Unit:  mp.UnitPercentage,
			Metrics: []mp.Metrics{
				{Name: "util", Label: "util"},
			},
		},
		"device.average.#": {
			Label: (labelPrefix + " Device Utilization - Average"),
			Unit:  mp.UnitFloat,
			Metrics: []mp.Metrics{
				{Name: "request_size", Label: "request size sectors"},
				{Name: "queue_length", Label: "queue length"},
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
				"request.reads", "merge.reads", "sector.read", "time.read",
				"request.writes", "merge.writes", "sector.written", "time.write",
				"inprogress.io", "time.io", "time.ioWeighted",
				"request.discards", "merge.discards", "sector.Discarded", "time.discard",
			}

			for i, metric := range matches[3:] {
				// TODO: Sanitize these values.
				key := fmt.Sprintf("disk.%s", strings.Replace(metricNames[i], ".", "."+device+".", 1))
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
