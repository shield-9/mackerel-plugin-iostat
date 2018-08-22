package mpiostat

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

type IostatPlugin struct {
	Prefix string
}

var iostatVersionHeaderPattern = regexp.MustCompile(
	`^Linux\s+.+\sCPU\)$`,
)

var iostatCpuHeaderPattern = regexp.MustCompile(
	`^avg-cpu:\s+`,
)

var iostatDeviceHeaderPattern = regexp.MustCompile(
	`^Device:\s+`,
)

var iostatCpuColumnsPattern = regexp.MustCompile(
	`^\s*(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s*$`,
)

var iostatDeviceColumnsPattern = regexp.MustCompile(
	`^(\S+)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s*$`,
)

func (i IostatPlugin) GraphDefinition() map[string]mp.Graphs {
	labelPrefix := strings.Title(i.MetricKeyPrefix())
	return map[string]mp.Graphs{
		"": {
			Label: labelPrefix,
			Unit:  mp.UnitFloat,
			Metrics: []mp.Metrics{
				{Name: "seconds", Label: "seconds"},
			},
		},
	}
}

/*
$ iostat -xk
Linux 3.10.0-862.3.2.el7.x86_64 (daisuke-tf-01.novalocal) 	08/22/18 	_x86_64_	(2 CPU)

avg-cpu:  %user   %nice %system %iowait  %steal   %idle
           0.08    0.00    0.04    0.00    0.00   99.87

Device:         rrqm/s   wrqm/s     r/s     w/s    rkB/s    wkB/s avgrq-sz avgqu-sz   await r_await w_await  svctm  %util
vda               0.00     0.03    0.44    0.22    23.15    13.73   112.51     0.00    1.56    0.71    3.24   0.41   0.03
vdb               0.00     0.00    0.01    0.00     0.22     0.00    47.27     0.00    0.32    0.32    0.00   0.20   0.00
*/

func (i IostatPlugin) FetchMetrics() (map[string]float64, error) {
	cmd := exec.Command("iostat", "-xk")
	cmd.Env = append(os.Environ(), "LANG=C")
	io, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("'iostat -xk' command exited with a non-zero status: %s", err)
	}
	result := make(map[string]float64)
	for _, line := range strings.Split(string(io), "\n") {
		if iostatVersionHeaderPattern.MatchString(line) || iostatCpuHeaderPattern.MatchString(line) || iostatDeviceHeaderPattern.MatchString(line) {
			continue
		} else if matches := iostatCpuColumnsPattern.FindStringSubmatch(line); matches != nil {
			//fmt.Printf("Cpu: %q\n", matches[1:])
		} else if matches := iostatDeviceColumnsPattern.FindStringSubmatch(line); matches != nil {
			//fmt.Printf("Dev: %q\n", matches[1:])
			device := matches[1]
			rrqmps, err := strconv.ParseFloat(matches[2], 64)
			wrqmps, err := strconv.ParseFloat(matches[3], 64)
			rps, err := strconv.ParseFloat(matches[4], 64)
			wps, err := strconv.ParseFloat(matches[5], 64)
			rkbps, err := strconv.ParseFloat(matches[6], 64)
			wkbps, err := strconv.ParseFloat(matches[7], 64)
			avgrq_sz, err := strconv.ParseFloat(matches[8], 64)
			avgqu_sz, err := strconv.ParseFloat(matches[9], 64)

			if err != nil {
				return nil, fmt.Errorf("Failed to parse value: %s", err)
			}

			result["device.request."+device+".read_merged"] = rrqmps
			result["device.request."+device+".write_merged"] = wrqmps
			result["device.request."+device+".read_completed"] = rps
			result["device.request."+device+".write_completed"] = wps
			result["device.transfer."+device+".read"] = rkbps
			result["device.transfer."+device+".write"] = wkbps
			result["device.request."+device+".avg_size"] = avgrq_sz
			result["device.request."+device+".avg_queue"] = avgqu_sz

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
