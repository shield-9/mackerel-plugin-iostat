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
	Prefix        string
	IgnoreVirtual bool
}

var deviceNamePattern = regexp.MustCompile(`[^[[:alnum:]]_-]`)

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
				// See https://github.com/torvalds/linux/blob/b219a1d2de0c025318475e3bbf8e3215cf49d083/Documentation/block/stat.txt#L50-L56 for details.
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

	blocks := make(map[string]bool)

	// Create list of virtual devices if required.
	if i.IgnoreVirtual {
		devices, err := i.fetchBlockdevices()
		if err != nil {
			return nil, err
		}
		blocks, err = i.analyzeBlockdevices(devices)
		if err != nil {
			return nil, err
		}
	}

	metrics := make(map[string]float64)
	for _, disk := range i.formatDiskstats(string(io)) {
		device := disk[2]

		// Skip if it's a virtual.
		if val, ok := blocks[device]; ok && !val {
			continue
		}

		deviceDispName := deviceNamePattern.ReplaceAllString(device, "")

		if err := i.parseStats(deviceDispName, disk, metrics); err != nil {
			return nil, err
		}
	}

	return metrics, nil
}

func (i IostatPlugin) MetricKeyPrefix() string {
	if i.Prefix == "" {
		i.Prefix = "disk"
	}
	return i.Prefix
}

func (i IostatPlugin) formatDiskstats(stats string) [][]string {
	result := [][]string{}

	for _, line := range strings.Split(stats, "\n") {
		matches := strings.Fields(line)

		// Skip for empty line. See https://github.com/golang/go/issues/13075 for details.
		if len(matches) == 0 || len(matches[0]) == 0 {
			continue
		}

		result = append(result, matches)
	}

	return result
}

func (i IostatPlugin) parseStats(label string, stats []string, metrics map[string]float64) error {
	var err error

	for i, metric := range stats[3:] {
		key := strings.Replace(metricNames[i], ".", "."+label+".", 1) // e.g. "time.io" => "time.vda1.io"
		metrics[key], err = strconv.ParseFloat(metric, 64)
		if err != nil {
			return fmt.Errorf("Failed to parse value: %s", err)
		}

		switch strings.Split(key, ".")[0] {
		case "request", "merge", "sector", "time":
			/*
				Mackerel is designed to display metrics in per-minute, while I want "per-second".
				\frac{(\frac{crntVal}{60} - \frac{lastVal}{60}) * 60}{crntTime - lastTime} = \frac{crntVal - lastVal}{crntTime - lastTime}
				See https://github.com/mackerelio/go-mackerel-plugin/blob/3980df9bc6311013061fb7ff66498ce23e275bdf/mackerel-plugin.go#L156 for details.
			*/
			metrics[key] /= 60
		}
	}

	return nil
}

func (i IostatPlugin) fetchBlockdevices() ([]os.FileInfo, error) {
	// Fetch list of block devices.
	devices, err := ioutil.ReadDir("/sys/block")
	if err != nil {
		return nil, fmt.Errorf("Cannot read from directory /sys/block/: %s", err)
	}

	return devices, nil
}

func (i IostatPlugin) analyzeBlockdevices(devices []os.FileInfo) (map[string]bool, error) {
	// Generate list of phyisical block devices to skip virtual ones, such as loopback.
	blocks := make(map[string]bool)
	for _, device := range devices {
		blocks[device.Name()] = false

		// Check if it's not a symlink.
		if device.Mode()&os.ModeSymlink != os.ModeSymlink {
			continue
		}

		real, err := os.Readlink(fmt.Sprintf("/sys/block/%s", device.Name()))
		if err != nil {
			return nil, fmt.Errorf("Cannot read from directory /sys/block/%s: %s", device.Name(), err)
		}

		// Check if it's a virtual device.
		if strings.HasPrefix(real, "../devices/virtual/block/") {
			continue
		}

		blocks[device.Name()] = true
	}

	return blocks, nil
}

func Do() {
	optPrefix := flag.String("metric-key-prefix", "disk", "Metric key prefix")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	optVirtual := flag.Bool("ignore-virtual", true, "Temp file name")
	flag.Parse()

	i := IostatPlugin{
		Prefix:        *optPrefix,
		IgnoreVirtual: *optVirtual,
	}
	plugin := mp.NewMackerelPlugin(i)
	plugin.Tempfile = *optTempfile
	plugin.Run()
}
