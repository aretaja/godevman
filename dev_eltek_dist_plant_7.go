package godevman

import "strings"

// Adds Eltek Distributed Plant v7 PSU specific SNMP functionality to snmpCommon type
type deviceEltekDP7 struct {
	snmpCommon
}

// Get running software version
func (sd *deviceEltekDP7) SwVersion() (string, error) {
	var out string
	res, err := sd.System([]string{"Descr"})
	if err != nil {
		return out, err
	}

	if res.Descr.IsSet {
		out = strings.TrimSpace(res.Descr.Value)
	}

	return out, nil
}
