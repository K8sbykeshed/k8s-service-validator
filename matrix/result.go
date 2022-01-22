package matrix

import (
	"fmt"
	"strconv"
	"strings"
)

// ProbeJobBandwidthResults models the results of a pod->pod connectivity bandwidth
type ProbeJobBandwidthResults struct {
	Bandwidth float64
}

// FromCommaSeparatedString parses the string output for iperf stdout
// sample line: 20220207193823,10.244.0.27,59654,10.244.0.27,80,3,0.0-10.0,127987744768,102389776016
func (r *ProbeJobBandwidthResults) FromCommaSeparatedString(s string) (err error) {
	splits := strings.Split(s, ",")
	r.Bandwidth, err = strconv.ParseFloat(splits[8], 32)
	return err
}

func prettyString(num float64, unit string) string {
	if num < 1e3 {
		return fmt.Sprintf("%f %s/sec", num, unit)
	} else if num < 1e6 {
		return fmt.Sprintf("%f K%s/sec", num/1e3, unit)
	} else if num < 1e9 {
		return fmt.Sprintf("%f M%s/sec", num/1e6, unit)
	} else {
		return fmt.Sprintf("%f G%s/sec", num/1e9, unit)
	}
}

// PrettyString prints human-readable string for bandwidth
func (r *ProbeJobBandwidthResults) PrettyString(inBytes bool) string {
	if inBytes {
		return prettyString(r.Bandwidth/8, "Bytes")
	}
	return prettyString(r.Bandwidth, "Bits")
}

// ProbeJobResults packages the model for the results of a pod->pod connectivity probe
type ProbeJobResults struct {
	Job         *ProbeJob
	IsConnected bool
	Err         error
	Command     string
	Endpoint    string
	Bandwidth   *ProbeJobBandwidthResults // nil if error or bandwidth is not required to measure
}
