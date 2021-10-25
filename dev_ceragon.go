package godevman

// Adds Ceragon specific SNMP functionality to snmpCommon type
type deviceCeragon struct {
	snmpCommon
}

// Get running software version
func (sd *deviceCeragon) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.2281.10.4.1.13.1.1.4.1"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}
