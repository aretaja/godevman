package godevman

// Adds Mikrotic specific SNMP functionality to DeviceSnmpGeneric type
type deviceMikrotik struct {
	snmpCommon
}

// Get running software version
func (sd *deviceMikrotik) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.14988.1.1.4.4.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}
