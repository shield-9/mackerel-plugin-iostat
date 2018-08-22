package mpiostat

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

type IostatPlugin struct {
	Prefix string
}

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

func (i IostatPlugin) FetchMetrics() (map[string]float64, error) {
	cmd := exec.Command("iostat", "-xk")
	cmd.Env = append(os.Environ(), "LANG=C")
	io, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("'iostat -xk' command exited with a non-zero status: %s", err)
	}
	return map[string]float64{"seconds": 0}, nil
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
