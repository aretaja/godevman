package godevman

import "net/http"

// Get system info
type DevSys interface {
	System(targets []string) (system, error)
	SetSysName(v string) error
	SetContact(v string) error
	SetLocation(v string) error
}

// Functionality related to interfaces
type DevIfs interface {
	IfInfo(targets []string, idx ...string) (map[string]*ifInfo, error)
	// Get ifNumber
	IfNumber() (int64, error)
	// Get interfaces stack info
	IfStack() (ifStack, error)
	// Set interface admin status
	SetIfAdmStat(set map[string]string) error
	// Set interface alias
	SetIfAlias(set map[string]string) error
}

// Functionality related to inventory
type DevInv interface {
	InvInfo(targets []string, idx ...string) (map[string]*invInfo, error)
	// Get interface to inventory relations
	IfInventory() (map[int]int, error)
}

// Functionality related to dot1q vlans
type DevVlan interface {
	D1qVlans() (map[string]string, error)
	// Get dot1q vlans
	BrPort2IfIdx() (map[string]int, error)
	// Get dot1q vlan to port relations
	D1qVlanInfo() (map[string]*d1qVlanInfo, error)
}

// Functionality related to IP addresses
type DevIp interface {
	IpInfo(targets []string, ip ...string) (map[string]*ipInfo, error)
	IpIfInfo(ip ...string) (map[string]*ipIfInfo, error)
}

// Functionality related to Get IPv6 addresses
type DevIp6 interface {
	Ip6IfDescr(idx ...string) (map[string]string, error)
}

// Get OSPF info
type DevOspf interface {
	OspfAreaRouters() (map[string][]string, error)
	OspfAreaStatus() (map[string]string, error)
	OspfNbrStatus() (map[string]string, error)
}

// Get Software version
type DevSw interface {
	SwVersion() (string, error)
}

// Functionality related to web connection authentication
type DevWebSess interface {
	WebAuth(userPass []string) error
	WebSession() *http.Client
	WebLogout() error
}

// Get RL neighbour info
type DevRl interface {
	RlInfo() (rlRadioInfo, error)
	RlNbrInfo() (rlRadioFeInfo, error)
}

// Get backup info
type DevBackupInfo interface {
	LastBackup() (backupInfo, error)
}
