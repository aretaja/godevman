// Package godevman - common godevman package
package godevman

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/aretaja/snmphelper"
	"github.com/kr/pretty"
)

// Version of release
const Version = "0.0.1"

// SNMP credentials for snmp session
type Snmpcred struct {
	Ver      int    // [snmp version] (1|2|3)
	User     string // [username|community]
	Prot     string // [authentication protocol] (NoAuth|MD5|SHA)
	Pass     string // [authentication protocol pass phrase]
	Slevel   string // [security level] (noAuthNoPriv|authNoPriv|authPriv)
	PrivProt string // [privacy protocol] (NoPriv|DES|AES|AES192|AES256|AES192C|AES256C)
	PrivPass string // [privacy protocol pass phrase]
}

// Parameters for new Device object initialization
type Dparams struct {
	Ip          string // ip of device
	Sysobjectid string // sysObjectId of Device
	Snmpcred    Snmpcred
}

// Device object
type device struct {
	ip          string              // ip of device
	sysname     string              // sysname of device
	sysobjectid string              // sysObjectId of device
	snmpsession *snmphelper.Session // snmp session of device
	// clisession  devicecli.Dcli   // cli session of device
	// websession  deviceweb.Dweb   // web session of device
	debug int // Debug level
}

// Initialize new device object
// func NewDevice(p *Dparams) (*device, error) {
func NewDevice(p *Dparams) (*device, error) {
	var d device
	// ip is required
	if net.ParseIP(p.Ip) == nil {
		return nil, fmt.Errorf("ip is required for new device object initialization")
	} else {
		d.ip = p.Ip
	}

	// Set Debug level if Env var is set
	if l, set := os.LookupEnv("GODEVMAN_DEBUG"); set {
		if li, err := strconv.Atoi(l); err != nil {
			return nil, fmt.Errorf("value of GODEVMAN_DEBUG must be integer")
		} else {
			d.debug = li
		}
	}

	// validate sysObjectId if defined
	if p.Sysobjectid != "" {
		if set, _ := regexp.Match(`^(no-snmp[-\w]*|\.[\.\d]+)$`, []byte(p.Sysobjectid)); set {
			d.sysobjectid = p.Sysobjectid
		} else {
			return nil, fmt.Errorf("not valid sysobjectid - %s", p.Sysobjectid)
		}
	}

	if set, _ := regexp.Match(`^no-snmp`, []byte(p.Sysobjectid)); !set && p.Snmpcred.User != "" {
		// Session variables
		session := snmphelper.Session{
			Host:     p.Ip,
			Ver:      p.Snmpcred.Ver,
			User:     p.Snmpcred.User,
			Prot:     p.Snmpcred.Prot,
			Pass:     p.Snmpcred.Pass,
			Slevel:   p.Snmpcred.Slevel,
			PrivProt: p.Snmpcred.PrivProt,
			PrivPass: p.Snmpcred.PrivPass,
		}

		// Initialize session
		sess, err := session.New()
		if err != nil {
			return nil, fmt.Errorf("create new snmp session failed - error: %v", err)
		}

		d.snmpsession = sess

		// get sysobjectid and sysname
		oids := map[string]string{"sysname": ".1.3.6.1.2.1.1.5.0"}
		if d.sysobjectid == "" {
			oids["sysobjectid"] = ".1.3.6.1.2.1.1.2.0"
		}

		o := make([]string, 0, len(oids))
		for _, oid := range oids {
			o = append(o, oid)
		}

		res, err := sess.Get(o)
		if err != nil {
			// HACK Eltek eNexus controller don't respond to sysObjectID query
			if strings.HasSuffix(err.Error(), "NoSuchObject") {
				_, err2 := sess.Get([]string{".1.3.6.1.4.1.12148.10.2.2.0"})
				if err2 == nil {
					d.sysobjectid = ".1.3.6.1.4.1.12148.10"
				}
			} else {
				return nil, fmt.Errorf("sysobjectid and sysname discovery failed - snmp error: %v", err)
			}
		} else {
			if val, ok := oids["sysobjectid"]; ok {
				soi := res[val].ObjectIdentifier
				// HACK Eaton UPS returns not appropriate sysobjectid
				if soi == ".2.1932768099.842208050.858927922.858993459.859026295.825438771.858993459" {
					soi = ".1.3.6.1.4.1.705.1"
				}
				d.sysobjectid = soi
			}
		}

		d.sysname = res[oids["sysname"]].OctetString
	}

	// DEBUG
	if d.debug > 0 {
		fmt.Printf("New device object: %# v\n", pretty.Formatter(d))
	}

	return &d, nil
}

// Morph - Type morphing according to device
func (d *device) Morph() interface{} {
	var res interface{} = d

	if strings.HasPrefix(d.sysobjectid, ".") {
		switch {
		case d.sysobjectid == ".1.3.6.1.4.1.2281.1.20.2.2.10" ||
			d.sysobjectid == ".1.3.6.1.4.1.2281.1.20.2.2.12" ||
			d.sysobjectid == ".1.3.6.1.4.1.2281.1.20.2.2.14":
			md := deviceCeragon{
				snmpCommon{*d},
			}
			res = &md
		case strings.HasPrefix(d.sysobjectid, ".1.3.6.1.4.1.9.1.1") ||
			strings.HasPrefix(d.sysobjectid, ".1.3.6.1.4.1.9.1.6"):
			md := deviceCisco{
				snmpCommon{*d},
			}
			res = &md
		case d.sysobjectid == ".1.3.6.1.4.1.12148.9":
			md := deviceEltekDP7{
				snmpCommon{*d},
			}
			res = &md
		case d.sysobjectid == ".1.3.6.1.4.1.12148.10":
			md := deviceEltekEnexus{
				snmpCommon{*d},
			}
			res = &md
		case d.sysobjectid == ".1.3.6.1.4.1.193.223.2.1":
			md := deviceEricssonMlPt{
				snmpCommon{*d},
			}
			res = &md
		case d.sysobjectid == ".1.3.6.1.4.1.193.81.1.1.1" ||
			d.sysobjectid == ".1.3.6.1.4.1.193.81.1.1.3":
			md := deviceEricssonMlTn{
				snmpCommon{*d},
			}
			res = &md
		case strings.HasPrefix(d.sysobjectid, ".1.3.6.1.4.1.2636.1.1.1.2"):
			md := deviceJuniper{
				snmpCommon{*d},
			}
			res = &md
		case d.sysobjectid == ".1.3.6.1.4.1.8072.3.2.10":
			sd := snmpCommon{*d}
			md := deviceLinux{sd}
			res = &md

			r, err := sd.System([]string{"Descr"})
			if err == nil {
				matched, _ := regexp.MatchString(`(?i)martem`, r.Descr.Value)
				if matched {
					md := deviceMartem{
						snmpCommon{*d},
					}
					res = &md
				}
			}

		case d.sysobjectid == ".1.3.6.1.4.1.14988.1":
			md := deviceMikrotik{
				snmpCommon{*d},
			}
			res = &md
		case d.sysobjectid == ".1.3.6.1.4.1.705.1" ||
			d.sysobjectid == ".1.3.6.1.4.1.534.1" ||
			d.sysobjectid == ".1.3.6.1.4.1.2254.2.4" ||
			d.sysobjectid == ".1.3.6.1.4.1.818.1.100.1.1":
			md := deviceUps{
				snmpCommon{*d},
			}
			res = &md
		default:
			res = &snmpCommon{*d}
		}
	}

	return res
}

// Common types for unified device info

// Common types for collected values
type valString struct {
	Value string
	IsSet bool
}

type valI64 struct {
	Value int64
	IsSet bool
}

type valU64 struct {
	Value uint64
	IsSet bool
}

// System data
type system struct {
	Descr, ObjectID, Contact, Name, Location, UpTimeStr valString
	UpTime                                              valU64
}

// Interface info
type ifInfo struct {
	Descr, Name, Alias, Mac, LastStr, TypeStr, AdminStr, OperStr valString
	Type, Mtu, Admin, Oper                                       valI64
	Speed, Last, InOctets, InUcast, InMcast, InBcast, InDiscards,
	InErrors, OutOctets, OutUcast, OutMast, OutBcast, OutDiscards,
	OutErrors valU64
}

// Interface stack info
type ifStack struct {
	Up, Down map[int][]int
}

// Inventory info
type invInfo struct {
	Physical bool
	ParentId valI64
	Descr, Position, HwProduct, HwRev, Serial, Manufacturer, Model, SwProduct,
	SwRev valString
}

// Dot1Q VLAN bridgeport info
type d1qVlanBrPort struct {
	IfIdx int
	UnTag bool
}

// Dot1Q VLAN info (Boolean in Ports map indicates untagged vlan)
type d1qVlanInfo struct {
	Name  string
	Ports map[int]*d1qVlanBrPort
}

// IP info
type ipInfo struct {
	IfIdx int64
	Mask  string
}

// IP Interface info
type ipIfInfo struct {
	ipInfo
	Descr, Alias string
}
