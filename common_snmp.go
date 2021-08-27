package godevman

import (
	"github.com/aretaja/snmphelper"
)

// Common SNMP functionality
type snmpCommon struct {
	device
}

// Get sysDescr
func (sd *snmpCommon) SysDescr() (string, error) {
	oid := ".1.3.6.1.2.1.1.1.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Get sysObjectID
func (sd *snmpCommon) SysObjectID() (string, error) {
	oid := ".1.3.6.1.2.1.1.2.0"
	r, err := sd.getone(oid)
	return r[oid].ObjectIdentifier, err
}

// Get sysUpTime - returns duration in milliseconds
func (sd *snmpCommon) SysUpTime() (uint64, error) {
	oid := ".1.3.6.1.2.1.1.3.0"
	r, err := sd.getone(oid)
	return r[oid].TimeTicks, err
}

// Get sysContact
func (sd *snmpCommon) SysContact() (string, error) {
	oid := ".1.3.6.1.2.1.1.4.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Get sysName
func (sd *snmpCommon) SysName() (string, error) {
	oid := ".1.3.6.1.2.1.1.5.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Get sysLocation
func (sd *snmpCommon) SysLocation() (string, error) {
	oid := ".1.3.6.1.2.1.1.6.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Get ifNumber
func (sd *snmpCommon) IfNumber() (int64, error) {
	oid := ".1.3.6.1.2.1.2.1.0"
	r, err := sd.getone(oid)
	return r[oid].Integer, err
}

// Single oid get helper
func (sd *snmpCommon) getone(oid string) (snmphelper.SnmpOut, error) {
	o := []string{oid}
	res, err := sd.snmpsession.Get(o)
	if err != nil {
		return nil, err
	}
	return res, nil
}
