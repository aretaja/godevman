package godevman

// Get system info
type DevSystem interface {
	System(targets []string) (system, error)
}

// Interfaces

// Get interface info
type DevIfInfo interface {
	IfInfo(targets []string, idx ...string) (map[string]*ifInfo, error)
	// Get ifNumber
	IfNumber() (int64, error)
	// Get interfaces stack info
	IfStack() (ifStack, error)
}

// Get inventory info
type DevInvInfo interface {
	InvInfo(targets []string, idx ...string) (map[string]*invInfo, error)
	// Get interface to inventory relations
	IfInventory() (map[int]int, error)
}

// Get dot1q vlan info
type DevVlanInfo interface {
	D1qVlans() (map[string]string, error)
	// Get dot1q vlans
	BrPort2IfIdx() (map[string]int, error)
	// Get dot1q vlan to port relations
	D1qVlanInfo() (map[string]*d1qVlanInfo, error)
}

// Get IP info
type DevIpInfo interface {
	IpInfo(targets []string, ip ...string) (map[string]*ipInfo, error)
	IpIfInfo(ip ...string) (map[string]*ipIfInfo, error)
}

// Get IPv6 info
type DevIp6Info interface {
	Ip6IfDescr(idx ...string) (map[string]string, error)
}

// Get OSPF info
type DevOspfInfo interface {
	OspfAreaRouters() (map[string][]string, error)
	OspfAreaStatus() (map[string]string, error)
	OspfNbrStatus() (map[string]string, error)
}
