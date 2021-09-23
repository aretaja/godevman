package godevman

import (
	"fmt"
	"regexp"
)

// Adds Cisco specific SNMP functionality to snmpCommon type
type deviceCisco struct {
	snmpCommon
}

// Get running software version
func (sd *deviceCisco) SwVersion() (string, error) {
	var out string
	res, err := sd.System([]string{"Descr"})
	if err != nil {
		return out, err
	}

	re := regexp.MustCompile(`Version (.*?)[,|\s]`)
	if res.Descr.IsSet {
		reMatch := re.FindStringSubmatch(res.Descr.Value)
		if reMatch == nil {
			return out, fmt.Errorf("failed to parse sysDescr for version - %s", res.Descr.Value)
		}
		out = reMatch[1]
	}

	return out, nil
}
