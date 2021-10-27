package godevman

// Adds Moxa specific SNMP functionality to snmpCommon type
type deviceMoxa struct {
	snmpCommon
}

// Get running software version
func (sd *deviceMoxa) SwVersion() (string, error) {
	oid := sd.sysobjectid + ".1.4.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}
