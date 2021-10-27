package godevman

// Adds Martem specific SNMP functionality to snmpCommon type
type deviceMartem struct {
	snmpCommon
}

// Get running software version
func (sd *deviceMartem) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.43098.2.1.4.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}
