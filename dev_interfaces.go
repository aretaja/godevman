package godevman

import "net/http"

// Get system info
type DevSysReader interface {
	System([]string) (System, error)
}

// Set system info
type DevSysWriter interface {
	SetSysName(string) error
	SetContact(string) error
	SetLocation(string) error
}

// Functionality related to interfaces
type DevIfReader interface {
	IfInfo([]string, ...string) (map[string]*IfInfo, error)
	// Get ifNumber
	IfNumber() (int64, error)
	// Get interfaces stack info
	IfStack() (IfStack, error)
}

type DevIfWriter interface {
	// Set interface admin status
	SetIfAdmStat(map[string]string) error
	// Set interface alias
	SetIfAlias(map[string]string) error
}

// Functionality related to inventory
type DevInvReader interface {
	InvInfo([]string, ...string) (map[string]*InvInfo, error)
	// Get interface to inventory relations
	IfInventory() (map[int]int, error)
}

// Functionality related to dot1q vlans
type DevVlanReader interface {
	D1qVlans() (map[string]string, error)
	// Get dot1q vlans
	BrPort2IfIdx() (map[string]int, error)
	// Get dot1q vlan to port relations
	D1qVlanInfo() (map[string]*D1qVlanInfo, error)
}

// Functionality related to IP addresses
type DevIpReader interface {
	IpInfo(...string) (map[string]*IpInfo, error)
	IpIfInfo(...string) (map[string]*IpIfInfo, error)
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

// Get Hardware info
type DevHwReader interface {
	HwInfo() (map[string]string, error)
}

// Functionality related to web connection authentication
type DevWebSessManager interface {
	WebAuth([]string) error
	WebSession() *http.Client
	WebLogout() error
}

// Get RL neighbour info
type DevRlReader interface {
	RlInfo() (map[string]*RlRadioIfInfo, error)
	RlNbrInfo() (map[string]*RlRadioFeIfInfo, error)
}

// Get backup info
type DevBackupReader interface {
	LastBackup() (*BackupInfo, error)
}

// Backup initiator
type DevBackupper interface {
	// Backup device config
	DoBackup() error
}

// Get environment sensors info
type DevSensorsReader interface {
	Sensors([]string) (map[string]map[string]map[string]SensorVal, error)
}

// Get ONU info
type DevOnusReader interface {
	OnuInfo() (map[string]*OnuInfo, error)
}

// Get Power Generator info
type DevGenReader interface {
	GeneratorInfo([]string) (GenInfo, error)
}

// Get energy readings
type DevEnergyMeterReader interface {
	Ereadings() (*EReadings, error)
}

// CLI releated functionality
type DevCliWriter interface {
	// Execute cli commands
	RunCmds([]string, *CliCmdOpts) ([]string, error)
}

// Get running config
type DevConfReader interface {
	RuningCfg() (string, error)
}

// Mobile signal related functionality
type DevMobReader interface {
	MobSignal() (map[string]MobSignal, error)
}

// Test interface
// type DevTest interface {
// 	TestCmd([]string) ([]string, error)
// }
