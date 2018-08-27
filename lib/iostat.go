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


func (i IostatPlugin) GraphDefinition() map[string]mp.Graphs {
	labelPrefix := strings.Title(i.MetricKeyPrefix())
	return map[string]mp.Graphs{
		"request.#": {
			Label: (labelPrefix + " Requests (/sec)"),
			Unit:  mp.UnitIOPS,
			Metrics: []mp.Metrics{
				{Name: "reads", Label: "read", Diff: true},
				{Name: "writes", Label: "write", Diff: true},
			},
		},
		"merge.#": {
			Label: (labelPrefix + " Merge (/sec)"),
			Unit:  mp.UnitFloat,
			Metrics: []mp.Metrics{
				{Name: "reads", Label: "read", Diff: true},
				{Name: "writes", Label: "write", Diff: true},
			},
		},
		"sector.#": {
			Label: (labelPrefix + " Traffic (sectors)"),
			Unit:  mp.UnitFloat,
			Metrics: []mp.Metrics{
				{Name: "read", Label: "read", Diff: true},
				{Name: "written", Label: "write", Diff: true},
			},
		},
		"time.#": {
			Label: (labelPrefix + " Time (ms/sec)"),
			Unit:  mp.UnitFloat,
			Metrics: []mp.Metrics{
				{Name: "read", Label: "read", Diff: true},
				{Name: "write", Label: "write", Diff: true},
				{Name: "io", Label: "io", Diff: true},
				{Name: "ioWeighted", Label: "io weighted", Diff: true},
			},
		},
		"inprogress.#": {
			Label: (labelPrefix + " IO in Progress"),
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "io", Label: "io"},
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

			deviceNamePattern := regexp.MustCompile(`[^[[:alnum:]]_-]`)
			device := deviceNamePattern.ReplaceAllString(matches[2], "")

			// TODO: Skip virtual devices, such as loop devices.

			// "Discard"s are introduced in Kernel 4.18. See linux/Documentation/iostats.txt for details.
			metricNames := []string{
				"request.reads", "merge.reads", "sector.read", "time.read",
				"request.writes", "merge.writes", "sector.written", "time.write",
				"inprogress.io", "time.io", "time.ioWeighted",
				"request.discards", "merge.discards", "sector.Discarded", "time.discard",
			}

			for i, metric := range matches[3:] {
				key := strings.Replace(metricNames[i], ".", "."+device+".", 1)
				result[key], err = strconv.ParseFloat(metric, 64)
				if err != nil {
					return nil, fmt.Errorf("Failed to parse value: %s", err)
				}

				switch strings.Split(key, ".")[0] {
				case "request", "merge", "sector", "time":
					/*
						Mackerel is designed to display metrics in per-minute, while I want "per-second".

						\frac{(\frac{crntVal}{60} - \frac{lastVal}{60}) * 60}{crntTime - lastTime} = \frac{crntVal - lastVal}{crntTime - lastTime}

						See https://github.com/mackerelio/go-mackerel-plugin/blob/3980df9bc6311013061fb7ff66498ce23e275bdf/mackerel-plugin.go#L156 for details.
					*/
					result[key] /= 60
				}
			}
		}
	}

	return result, nil
}

func (i IostatPlugin) MetricKeyPrefix() string {
	if i.Prefix == "" {
		i.Prefix = "disk"
	}
	return i.Prefix
}

func Do() {
	optPrefix := flag.String("metric-key-prefix", "disk", "Metric key prefix")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	i := IostatPlugin{
		Prefix: *optPrefix,
	}
	plugin := mp.NewMackerelPlugin(i)
	plugin.Tempfile = *optTempfile
	plugin.Run()
}
