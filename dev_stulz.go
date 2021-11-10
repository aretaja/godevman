package godevman

import "strings"

// Adds Stulz specific SNMP functionality to snmpCommon type
type deviceStulz struct {
	snmpCommon
}

// Get running software version
func (sd *deviceStulz) SwVersion() (string, error) {
	if strings.HasSuffix(sd.sysObjectId, ".29462.10") {
		oid := ".1.3.6.1.4.1.29462.10.1.1.1.65540.0"
		r, err := sd.getone(oid)
		return r[oid].OctetString, err
	} else {
		r, err := sd.System([]string{"Descr"})
		if err != nil {
			return "", err
		}

		return r.Descr.Value, err
	}
}
