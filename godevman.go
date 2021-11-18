// Package godevman - common godevman package
package godevman

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/aretaja/snmphelper"
	"github.com/davecgh/go-spew/spew"
)

// Version of release
const Version = "0.0.1"

// SNMP credentials for snmp session
type SnmpCred struct {
	User     string // [username|community]
	Prot     string // [authentication protocol] (NoAuth|MD5|SHA)
	Pass     string // [authentication protocol pass phrase]
	Slevel   string // [security level] (noAuthNoPriv|authNoPriv|authPriv)
	PrivProt string // [privacy protocol] (NoPriv|DES|AES|AES192|AES256|AES192C|AES256C)
	PrivPass string // [privacy protocol pass phrase]
	Ver      int    // [snmp version] (1|2|3)
}

// Parameters for new Device object initialization
type Dparams struct {
	Ip          string // ip of device
	SysObjectId string // sysObjectId of Device
	SnmpCred    SnmpCred
	Webcred     []string // Websession credentials
}

// Websession parameters
type webSess struct {
	client *http.Client // web client of device
	cred   []string     // web session credentials
}

// Device object
type device struct {
	snmpSession *snmphelper.Session // snmp session of device
	// clisession  devicecli.Dcli   // cli session of device
	webSession  *webSess // web session of device
	ip          string   // ip of device
	sysName     string   // sysname of device
	sysObjectId string   // sysObjectId of device
	debug       int      // Debug level
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

	// Setup Web session data
	d.webSession = &webSess{
		client: nil,
		cred:   nil,
	}
	if p.Webcred != nil {
		d.webSession.cred = p.Webcred
	}

	// validate sysObjectId if defined
	if p.SysObjectId != "" {
		if set, _ := regexp.Match(`^(no-snmp[-\w]*|\.[\.\d]+)$`, []byte(p.SysObjectId)); set {
			d.sysObjectId = p.SysObjectId
		} else {
			return nil, fmt.Errorf("not valid sysobjectid - %s", p.SysObjectId)
		}
	}

	if set, _ := regexp.Match(`^no-snmp`, []byte(p.SysObjectId)); !set && p.SnmpCred.User != "" {
		// Session variables
		session := snmphelper.Session{
			Host:     p.Ip,
			Ver:      p.SnmpCred.Ver,
			User:     p.SnmpCred.User,
			Prot:     p.SnmpCred.Prot,
			Pass:     p.SnmpCred.Pass,
			Slevel:   p.SnmpCred.Slevel,
			PrivProt: p.SnmpCred.PrivProt,
			PrivPass: p.SnmpCred.PrivPass,
		}

		// Initialize SNMP session
		sess, err := session.New()
		if err != nil {
			return nil, fmt.Errorf("create new snmp session failed - error: %v", err)
		}

		d.snmpSession = sess

		// get sysobjectid and sysname
		oids := map[string]string{"sysname": ".1.3.6.1.2.1.1.5.0"}
		if d.sysObjectId == "" {
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
					d.sysObjectId = ".1.3.6.1.4.1.12148.10"
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
				d.sysObjectId = soi
			}
		}

		d.sysName = res[oids["sysname"]].OctetString
	}

	// DEBUG
	if d.debug > 0 {
		spew.Printf("New device object: %# v\n", d)
	}

	return &d, nil
}

// Morph - Type morphing according to device
func (d *device) Morph() interface{} {
	var res interface{} = d

	if strings.HasPrefix(d.sysObjectId, ".") {
		// HACK for broken SNMP implementation in STULZ WIB1000 devices
		if d.sysObjectId == "0.0" {
			_, err := d.snmpSession.Get([]string{".1.3.6.1.4.1.39983.1.1.1.1.0"})
			if err == nil {
				d.sysObjectId = "1.3.6.1.4.1.39983.1.1"
			}
		}

		switch {
		case d.sysObjectId == ".1.3.6.1.4.1.2281.1.20.2.2.10" ||
			d.sysObjectId == ".1.3.6.1.4.1.2281.1.20.2.2.12" ||
			d.sysObjectId == ".1.3.6.1.4.1.2281.1.20.2.2.14":
			md := deviceCeragon{
				snmpCommon{*d},
			}
			res = &md
		case strings.HasPrefix(d.sysObjectId, ".1.3.6.1.4.1.9.1.1") ||
			strings.HasPrefix(d.sysObjectId, ".1.3.6.1.4.1.9.1.6"):
			md := deviceCisco{
				snmpCommon{*d},
			}
			res = &md
		case d.sysObjectId == ".1.3.6.1.4.1.12148.9":
			md := deviceEltekDP7{
				snmpCommon{*d},
			}
			res = &md
		case d.sysObjectId == ".1.3.6.1.4.1.12148.10":
			md := deviceEltekEnexus{
				snmpCommon{*d},
			}
			res = &md
		case d.sysObjectId == ".1.3.6.1.4.1.193.223.2.1":
			md := deviceEricssonMlPt{
				snmpCommon{*d},
			}
			res = &md
		case d.sysObjectId == ".1.3.6.1.4.1.193.81.1.1.1" ||
			d.sysObjectId == ".1.3.6.1.4.1.193.81.1.1.3":
			md := deviceEricssonMlTn{
				snmpCommon{*d},
			}
			res = &md
		case strings.HasPrefix(d.sysObjectId, ".1.3.6.1.4.1.2636.1.1.1.2"):
			md := deviceJuniper{
				snmpCommon{*d},
			}
			res = &md
		case d.sysObjectId == ".1.3.6.1.4.1.8072.3.2.10":
			sd := snmpCommon{*d}
			md := deviceLinux{sd}
			res = &md

			r, err := sd.System([]string{"Descr"})
			if err == nil {
				if match, _ := regexp.MatchString(`(?i)martem`, r.Descr.Value); match {
					md := deviceMartem{
						snmpCommon{*d},
					}
					res = &md
				}
			}
		case d.sysObjectId == ".1.3.6.1.4.1.14988.1":
			md := deviceMikrotik{
				snmpCommon{*d},
			}
			res = &md
		case strings.HasPrefix(d.sysObjectId, ".1.3.6.1.4.1.8691.7"):
			md := deviceMoxa{
				snmpCommon{*d},
			}
			res = &md
		case d.sysObjectId == ".1.3.6.1.4.1.2606.7":
			md := deviceRittal{
				snmpCommon{*d},
			}
			res = &md
		case d.sysObjectId == ".1.3.6.1.4.1.15004.2.1":
			md := deviceRuggedcom{
				snmpCommon{*d},
			}
			res = &md
		case d.sysObjectId == ".1.3.6.1.4.1.39983.1.1" ||
			d.sysObjectId == ".1.3.6.1.4.1.29462.10":
			md := deviceStulz{
				snmpCommon{*d},
			}
			res = &md
		case d.sysObjectId == ".1.3.6.1.4.1.41112.1.5":
			md := deviceUbiquiti{
				snmpCommon{*d},
			}
			res = &md
		case d.sysObjectId == ".1.3.6.1.4.1.705.1" ||
			d.sysObjectId == ".1.3.6.1.4.1.534.1" ||
			d.sysObjectId == ".1.3.6.1.4.1.2254.2.4" ||
			d.sysObjectId == ".1.3.6.1.4.1.818.1.100.1.1":
			md := deviceUps{
				snmpCommon{*d},
			}
			res = &md
		case d.sysObjectId == ".1.3.6.1.4.1.13858":
			md := deviceValere{
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
type valBool struct {
	Value bool
	IsSet bool
}

type valF64 struct {
	Value float64
	IsSet bool
}

type valInt struct {
	Value int
	IsSet bool
}

type valI64 struct {
	Value int64
	IsSet bool
}

type valString struct {
	Value string
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
	Descr, Position, HwProduct, HwRev, Serial, Manufacturer, Model, SwProduct,
	SwRev valString
	Physical bool
	ParentId valI64
}

// Dot1Q VLAN bridgeport info
type d1qVlanBrPort struct {
	IfIdx int
	UnTag bool
}

// Dot1Q VLAN info (Boolean in Ports map indicates untagged vlan)
type d1qVlanInfo struct {
	Ports map[int]*d1qVlanBrPort
	Name  string
}

// IP info
type ipInfo struct {
	Mask  string
	IfIdx int64
}

// IP Interface info
type ipIfInfo struct {
	Descr, Alias string
	ipInfo
}

// last backup info
type backupInfo struct {
	TargetIP, TargetFile string
	Timestamp, Progress  int
	Success              bool
}

// Radiolink radio interface info
type rfInfo struct {
	Name       valString
	Descr      valString
	Status     valString
	Mute       valBool
	IfIdx      valInt
	EntityIdx  valInt
	TxCapacity valInt
	PowerIn    valF64
	PowerOut   valF64
	Snr        valF64
}

type rauInfo struct {
	Rf        map[string]*rfInfo
	Name      valString
	Descr     valString
	EntityIdx valInt
	Temp      valF64
}

type rlRadioIfInfo struct {
	Rau       map[string]*rauInfo
	Name      valString
	Descr     valString
	AdmStat   valString
	OperStat  valString
	IfIdx     valInt
	EntityIdx valInt
	Es        valInt
	Uas       valInt
}

/*
// Radiolink radio info
type rlRadioInfo struct {
	Interfaces map[string]rlRadioIfInfo
}
*/
// Radiolink FarEnd radio interface info
type rlRadioFeIfInfo struct {
	SysName    valString
	Ip         valString
	IfIdx      valInt
	EntityIdx  valInt
	TxCapacity valInt
	PowerIn    valF64
	PowerOut   valF64
}

// Radiolink radio info
type rlRadioFeInfo struct {
	Neighbrs map[string]rlRadioFeIfInfo
}
