package godevman

// Adds generic UPS SNMP functionality to snmpCommon type
type deviceUps struct {
	snmpCommon
}

// Get running software version
func (sd *deviceUps) SwVersion() (string, error) {
	oid := ".1.3.6.1.2.1.33.1.1.4.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}
