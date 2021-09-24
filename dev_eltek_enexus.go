package godevman

import "strings"

// Adds Eltek eNexus PSU specific SNMP functionality to snmpCommon type
type deviceEltekEnexus struct {
	snmpCommon
}

// Get info from .iso.org.dod.internet.private.enterprises.eltek.eNexus.powerSystem tree
// Replaces common snmp method
// Valid targets values: "All", "Descr", "ObjectID", "UpTime", "Contact", "Name", "Location"
func (sd *deviceEltekEnexus) System(targets []string) (system, error) {
	var out system
	var idx []string

	for _, t := range targets {
		switch t {
		case "All":
			idx = []string{"4.0", "5.0", "6.0"}
			continue
		case "Contact":
			idx = append(idx, "4.0")
		case "Location":
			idx = append(idx, "5.0")
		case "Descr":
			idx = append(idx, "6.0")
		}
	}

	oid := ".1.3.6.1.4.1.12148.10.2"
	r, err := sd.getmulti(oid, idx)
	if err != nil {
		return out, err
	}

	for o, d := range r {
		switch o {
		case oid + ".4.0":
			out.Contact.Value = d.OctetString
			out.Contact.IsSet = true
		case oid + ".5.0":
			out.Location.Value = d.OctetString
			out.Location.IsSet = true
		case oid + ".6.0":
			out.Descr.Value = strings.TrimSpace(d.OctetString)
			out.Descr.IsSet = true
		}
	}

	out.ObjectID.Value = ".1.3.6.1.4.1.12148.10"
	out.ObjectID.IsSet = true
	return out, err
}

// Get running software version
func (sd *deviceEltekEnexus) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.12148.10.13.8.2.1.8.1"
	r, err := sd.getone(oid)
	return strings.TrimSpace(r[oid].OctetString), err
}
