package godevman

// Adds Ubiquiti specific SNMP functionality to snmpCommon type
type deviceUbiquiti struct {
	snmpCommon
}

// Get running software version
func (sd *deviceUbiquiti) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.41112.1.5.1.3.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}
