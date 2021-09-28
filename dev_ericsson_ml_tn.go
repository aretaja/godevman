package godevman

import "strings"

// Adds Ericsson MINI-LINK TN specific SNMP functionality to snmpCommon type
type deviceEricssonMlTn struct {
	snmpCommon
}

// Get running software version
func (sd *deviceEricssonMlTn) SwVersion() (string, error) {
	if sd.sysobjectid == ".1.3.6.1.4.1.193.81.1.1.1" { // Compact Node
		oid := ".1.3.6.1.4.1.193.81.2.7.1.1.1.4.1.1"
		r, err := sd.getone(oid)
		return strings.TrimSpace(r[oid].OctetString), err
	}

	// get installed sw states
	oid := ".1.3.6.1.4.1.193.81.2.7.1.2.1.5"
	r, err := sd.snmpsession.Walk(oid, true, true)
	if err != nil && sd.handleErr(oid, err) {
		return "", err
	}

	for k, v := range r {
		if v.Integer == 7 {
			oid = ".1.3.6.1.4.1.193.81.2.7.1.2.1.3." + k
			r, err := sd.getone(oid)
			return strings.TrimSpace(r[oid].OctetString), err
		}
	}

	return "", err
}
