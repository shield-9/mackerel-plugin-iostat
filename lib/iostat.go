package mpiostat

import (
	"flag"
	"fmt"
	"strings"

	mp "github.com/mackerelio/go-mackerel-plugin"
	"github.com/mackerelio/go-osstat/uptime"
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
	ut, err := uptime.Get()
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch uptime metrics: %s", err)
	}
	return map[string]float64{"seconds": ut.Seconds()}, nil
}

func (i IostatPlugin) MetricKeyPrefix() string {
	if i.Prefix == "" {
		i.Prefix = "uptime"
	}
	return i.Prefix
}

func Do() {
	optPrefix := flag.String("metric-key-prefix", "uptime", "Metric key prefix")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	i := IostatPlugin{
		Prefix: *optPrefix,
	}
	plugin := mp.NewMackerelPlugin(i)
	plugin.Tempfile = *optTempfile
	plugin.Run()
}
