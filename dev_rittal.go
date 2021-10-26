package godevman

// Adds Rittal specific SNMP functionality to snmpCommon type
type deviceRittal struct {
	snmpCommon
}

// Get running software version
func (sd *deviceRittal) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.2606.7.2.4.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}
