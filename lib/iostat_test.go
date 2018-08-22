package mpiostat

import "testing"

func TestFetchMetrics(t *testing.T) {
	iostat := &IostatPlugin{}

	ret, err := iostat.FetchMetrics()

	if err != nil {
		t.Errorf("FetchMetrics returns error")
	}

	if !(0 < ret["seconds"]) {
		t.Errorf("FetchMetrics doesn't return a value greater than 0")
	}
}
