package mpiostat

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

type IostatPlugin struct {
	Prefix string
}

// "Discard"s are introduced in Kernel 4.18. See linux/Documentation/iostats.txt for details.
var metricNames = []string{
	"request.reads", "merge.reads", "sector.read", "time.read",
	"request.writes", "merge.writes", "sector.written", "time.write",
	"inprogress.io", "time.io", "time.ioWeighted",
	"request.discards", "merge.discards", "sector.Discarded", "time.discard",
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
			Label: (labelPrefix + " Traffic"),
			Unit:  mp.UnitBytesPerSecond,
			Metrics: []mp.Metrics{
				// 1 sector is fixed to 512 bytes in Linux system.
				// See https://github.com/torvalds/linux/blob/b219a1d2de0c025318475e3bbf8e3215cf49d083/Documentation/block/stat.txt#L50 for details.
				{Name: "read", Label: "read", Scale: 2, Diff: true},
				{Name: "written", Label: "write", Scale: 2, Diff: true},
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

	blocks, err := i.fetchBlockdevices()
	if err != nil {
		return nil, err
	}

	result := make(map[string]float64)
	for _, line := range strings.Split(string(io), "\n") {
		matches := strings.Fields(line)

		// Skip for empty line. See https://github.com/golang/go/issues/13075 for details.
		if len(matches) == 0 || len(matches[0]) == 0 {
			continue
		}

		device := matches[2]

		// Skip if it's a virtual.
		if val, ok := blocks[device]; ok && !val {
			continue
		}

		deviceNamePattern := regexp.MustCompile(`[^[[:alnum:]]_-]`)
		deviceDispName := deviceNamePattern.ReplaceAllString(device, "")

		for i, metric := range matches[3:] {
			key := strings.Replace(metricNames[i], ".", "."+deviceDispName+".", 1)
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

	return result, nil
}

func (i IostatPlugin) MetricKeyPrefix() string {
	if i.Prefix == "" {
		i.Prefix = "disk"
	}
	return i.Prefix
}

func (i IostatPlugin) fetchBlockdevices() (map[string]bool, error) {
	// Fetch list of block devices.
	_blocks, err := ioutil.ReadDir("/sys/block")
	if err != nil {
		return nil, fmt.Errorf("Cannot read from directory /sys/block/: %s", err)
	}

	// Generate list of phyisical block devices to skip virtual ones, such as loopback.
	blocks := make(map[string]bool)
	for _, block := range _blocks {
		blocks[block.Name()] = false

		// Check if it's not a symlink.
		if block.Mode()&os.ModeSymlink != os.ModeSymlink {
			continue
		}

		real, err := os.Readlink(fmt.Sprintf("/sys/block/%s", block.Name()))
		if err != nil {
			return nil, fmt.Errorf("Cannot read from directory /sys/block/%s: %s", block.Name(), err)
		}

		// Check if it's a virtual device.
		if strings.HasPrefix(real, "../devices/virtual/block/") {
			continue
		}

		blocks[block.Name()] = true
	}

	return blocks, nil
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
