package godevman

import "strings"

// Adds Linux specific SNMP functionality to snmpCommon type
type deviceLinux struct {
	snmpCommon
}

// Get running software version
func (sd *deviceLinux) SwVersion() (string, error) {
	// find index for kernel uname if any (must be configured in device snmpd.conf)
	oid := ".1.3.6.1.4.1.8072.1.3.2.2.1.2"
	r, err := sd.snmpsession.Walk(oid, true, true)
	if err != nil && sd.handleErr(oid, err) {
		return "", err
	}

	var token string
	for k, v := range r {
		if strings.HasSuffix(v.OctetString, "/uname") {
			token = k
			break
		}
	}

	version := "Na"
	if len(token) != 0 {
		oid = ".1.3.6.1.4.1.8072.1.3.2.3.1.4." + token
		r, err = sd.getone(oid)
		if err != nil && sd.handleErr(oid, err) {
			return version, err
		}

		if r[oid].Vtype == "Integer" && r[oid].Integer == 0 {
			oid = ".1.3.6.1.4.1.8072.1.3.2.3.1.1." + token
			r, err = sd.getone(oid)
			if err != nil && sd.handleErr(oid, err) {
				return version, err
			}

			version = r[oid].OctetString
		}
	}

	return version, err
}
