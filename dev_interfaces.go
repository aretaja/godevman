package godevman

// System

// Get sysDescr
type DevSysDescr interface {
	SysDescr() (string, error)
}

// Get sysObjectID
type DevSysObjectID interface {
	SysObjectID() (string, error)
}

// Get sysUpTime
type DevSysUpTime interface {
	SysUpTime() (uint64, error)
}

// Get sysContact
type DevSysContact interface {
	SysContact() (string, error)
}

// Get sysName
type DevSysName interface {
	SysName() (string, error)
}

// Get sysLocation
type DevSysLocation interface {
	SysLocation() (string, error)
}

// Interfaces

// Get ifNumber
type DevIfNumber interface {
	IfNumber() (int64, error)
}

// Get ifDescr
type DevIfDescr interface {
	IfDescr(idx ...string) (map[string]string, error)
}