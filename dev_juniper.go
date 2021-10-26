package godevman

// Adds Juniper specific SNMP functionality to snmpCommon type
type deviceJuniper struct {
	snmpCommon
}

// Get running software version
func (sd *deviceJuniper) SwVersion() (string, error) {
	oid := ".1.3.6.1.2.1.25.6.3.1.2.2"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}
