package godevman

import "net/http"

// Get system info
type DevSysReader interface {
	System([]string) (system, error)
}

// Set system info
type DevSysWriter interface {
	SetSysName(string) error
	SetContact(string) error
	SetLocation(string) error
}

// Functionality related to interfaces
type DevIfReader interface {
	IfInfo([]string, ...string) (map[string]*ifInfo, error)
	// Get ifNumber
	IfNumber() (int64, error)
	// Get interfaces stack info
	IfStack() (ifStack, error)
}

type DevIfWriter interface {
	// Set interface admin status
	SetIfAdmStat(map[string]string) error
	// Set interface alias
	SetIfAlias(map[string]string) error
}

// Functionality related to inventory
type DevInvReader interface {
	InvInfo([]string, ...string) (map[string]*invInfo, error)
	// Get interface to inventory relations
	IfInventory() (map[int]int, error)
}

// Functionality related to dot1q vlans
type DevVlanReader interface {
	D1qVlans() (map[string]string, error)
	// Get dot1q vlans
	BrPort2IfIdx() (map[string]int, error)
	// Get dot1q vlan to port relations
	D1qVlanInfo() (map[string]*d1qVlanInfo, error)
}

// Functionality related to IP addresses
type DevIpReader interface {
	IpInfo(...string) (map[string]*ipInfo, error)
	IpIfInfo(...string) (map[string]*ipIfInfo, error)
}

// Functionality related to Get IPv6 addresses
type DevIp6Reader interface {
	Ip6IfDescr(...string) (map[string]string, error)
}

// Get OSPF info
type DevOspfReader interface {
	OspfAreaRouters() (map[string][]string, error)
	OspfAreaStatus() (map[string]string, error)
	OspfNbrStatus() (map[string]string, error)
}

// Get Software version
type DevSwReader interface {
	SwVersion() (string, error)
}

// Functionality related to web connection authentication
type DevWebSessManager interface {
	WebAuth([]string) error
	WebSession() *http.Client
	WebLogout() error
}

// Get RL neighbour info
type DevRlReader interface {
	RlInfo() (map[string]*rlRadioIfInfo, error)
	RlNbrInfo() (map[string]*rlRadioFeIfInfo, error)
}

// Get backup info
type DevBackupReader interface {
	LastBackup() (*backupInfo, error)
}

// Backup initiator
type DevBackupper interface {
	// Backup device config
	DoBackup() error
}

// Get environment sensors info
type DevSensorsReader interface {
	Sensors([]string) (map[string]map[string]map[string]sensorVal, error)
}

// Get ONU info
type DevOnusReader interface {
	OnuInfo() (map[string]*onuInfo, error)
}

// Get energy readings
type DevEnergyMeterReader interface {
	Ereadings() (*eReadings, error)
}

// CLI releated functionality
type DevCliWriter interface {
	// Execute cli commands
	RunCmds([]string, bool) ([]string, error)
}

// Get running config
type DevConfReader interface {
	RuningCfg() (string, error)
}

// Test interface
// type DevTest interface {
// 	TestCmd([]string) ([]string, error)
// }
