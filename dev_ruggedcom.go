package godevman

// Adds Ruggedcom specific SNMP functionality to snmpCommon type
type deviceRuggedcom struct {
	snmpCommon
}

// Get running software version
func (sd *deviceRuggedcom) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.15004.4.2.3.3.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}
