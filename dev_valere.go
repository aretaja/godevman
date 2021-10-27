package godevman

// Adds Valere specific SNMP functionality to snmpCommon type
type deviceValere struct {
	snmpCommon
}

// Get running software version
func (sd *deviceValere) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.13858.2.1.3.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}
