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
	"time"

	"github.com/aretaja/snmphelper"
	"github.com/davecgh/go-spew/spew"
	expect "github.com/google/goexpect"
	"github.com/patrickmn/go-cache"
)

// Version of release
const Version = "0.0.1"

const time_iso8601_sec = "2006-01-02T15:04:05"

// SNMP credentials for snmp session
type SnmpCred struct {
	// [username|community]
	User string
	// [authentication protocol] (NoAuth|MD5|SHA)
	Prot string
	// [authentication protocol pass phrase]
	Pass string
	// [security level] (noAuthNoPriv|authNoPriv|authPriv)
	Slevel string
	// [privacy protocol] (NoPriv|DES|AES|AES192|AES256|AES192C|AES256C)
	PrivProt string
	// [privacy protocol pass phrase]
	PrivPass string
	// [snmp version] (1|2|3)
	Ver int
}

// CLI session parameters
type CliParams struct {
	// Prompt re pattern
	// Default depends on device type
	// Keep it as is if you are not sure
	PromptRe string
	// Cli errors re pattern
	// Default depends on device type
	// Keep it as is if you are not sure
	ErrRe string
	// Default "22" for ssh and "23" for telnet
	Port string
	// Full path to unencrypted PEM-encoded private key file
	// If declared PublicKeys method for remote authentication will be used
	KeyPath string
	// Private key passphrase if any
	KeySecret string
	// Valid values are "\r\n" or "\n"
	// Default depends on device type
	// Keep it as is if you are not sure
	LineEnd string
	// CLI session credentials
	Cred []string
	// Commands which will be executed first on session start
	// Default depends on device type
	// Keep it as is if you are not sure
	PreCmds []string
	// Commands which will be executed to gracefully close session
	// Default depends on device type
	// Keep it as is if you are not sure
	DisconnectCmds []string
	// Use telnet instead of ssh. Default false
	Telnet bool
	// Session timeout (sec)
	// Depends on device type
	// Keep it as is if you are not sure
	Timeout int
}

// CLI command exec options
type CliCmdOpts struct {
	// Parse errors from cli output after submitting commands
	// Default false
	ChkErr bool
	// Run commands in privilrged mode (applicable on some device types)
	// Default false
	Priv bool
}

// Info needed for Device backup
type BackupParams struct {
	// IP of backup target system
	TargetIp string
	// Device identifier. Default is device ip
	// Will be used as first part of backup file name
	DevIdent string
	// Base path for backups (if device type needs it)
	BasePath string
	// Credentials
	Cred []string
}

// Parameters for new Device object initialization
type Dparams struct {
	// ip of device
	Ip string
	// sysObjectId of Device
	// Will be not discovered If present
	SysObjectId string
	// Timezone for time related actions
	// Default is Europe/Tallinn
	TimeZone string
	// Websession credentials
	WebCred      []string
	BackupParams BackupParams
	SnmpCred     SnmpCred
	CliParams    CliParams
}

// Websession
type webSess struct {
	// web client of device
	client *http.Client
	// web session credentials
	cred []string
}

// Clisession
type cliSess struct {
	// cli expecter of device
	client *expect.GExpect
	// cli session parameters
	params *CliParams
}

// Device object
type device struct {
	// snmp session of device
	snmpSession *snmphelper.Session
	// web session data of device
	webSession *webSess
	// cli session data of device
	cliSession *cliSess
	// Cache object
	cache *cache.Cache
	// Backup parameters
	backupParams *BackupParams
	// ip of device
	ip string
	// sysname of device
	sysName string
	// sysObjectId of device
	sysObjectId string
	// timezone related actions will use this.
	timeZone string
	// Debug level
	debug int
	// Enable use of cache
	useCache bool
}

// Initialize new device object
// func NewDevice(p *Dparams) (*device, error) {
func NewDevice(p *Dparams) (*device, error) {
	var d device
	// ip is required
	if net.ParseIP(p.Ip) == nil {
		return nil, fmt.Errorf("ip is required for new device object initialization")
	}
	d.ip = p.Ip

	// Set Debug level if Env var is set
	if l, set := os.LookupEnv("GODEVMAN_DEBUG"); set {
		if li, err := strconv.Atoi(l); err != nil {
			return nil, fmt.Errorf("value of GODEVMAN_DEBUG must be integer")
		} else {
			d.debug = li
		}
	}

	// Setup Web session data
	d.webSession = new(webSess)
	if p.WebCred != nil {
		d.webSession.cred = p.WebCred
	}

	// Setup CLI session data
	d.cliSession = new(cliSess)
	if p.CliParams.Cred != nil {
		d.cliSession.params = &p.CliParams
	}

	// Setup Backup info parameters
	d.backupParams = &p.BackupParams
	if p.BackupParams.DevIdent == "" {
		d.backupParams.DevIdent = d.ip
	}

	d.timeZone = "Europe/Tallinn"
	if p.TimeZone != "" {
		_, err := time.LoadLocation(p.TimeZone)
		if err == nil {
			d.timeZone = p.TimeZone
		}
	}

	// validate sysObjectId if defined
	if p.SysObjectId != "" {
		if set, _ := regexp.Match(`^(no-snmp[-\w]*|\.[\.\d]+)$`, []byte(p.SysObjectId)); set {
			d.sysObjectId = p.SysObjectId
		} else {
			return nil, fmt.Errorf("not valid sysobjectid - %s", p.SysObjectId)
		}
	}

	if set := strings.HasPrefix(p.SysObjectId, "no-snmp"); !set && p.SnmpCred.User != "" {
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

		// Don't do any snmp communication if sysObjectId is present
		if p.SysObjectId == "" {
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
	}

	// Setup cache
	d.cache = cache.New(10*time.Second, 10*time.Second)
	d.useCache = true

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
		if d.sysObjectId == ".0.0" {
			_, err := d.snmpSession.Get([]string{".1.3.6.1.4.1.39983.1.1.1.1.0"})
			if err == nil {
				d.sysObjectId = ".1.3.6.1.4.1.39983.1.1"
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
		case strings.HasPrefix(d.sysObjectId, ".1.3.6.1.4.1.9.1.") ||
			strings.HasPrefix(d.sysObjectId, ".1.3.6.1.4.1.9.6."):
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
		case strings.HasPrefix(d.sysObjectId, ".1.3.6.1.4.1.2636.1.1.1.2."):
			md := deviceJuniper{
				snmpCommon{*d},
			}
			res = &md
		case d.sysObjectId == ".1.3.6.1.4.1.8072.3.2.10":
			sd := snmpCommon{*d}
			md := deviceLinux{sd}
			res = &md

			r, err := md.System([]string{"Descr"})
			if err == nil {
				bOpts, _ := md.BuildOpts()
				vType, vErr := sd.getone(".1.3.6.1.4.1.12578.3.2.1.1.1.0")

				martemRe := regexp.MustCompile(`(?i)martem`)
				violaRe := regexp.MustCompile(`(?i)viola`)
				violaHttpRe := regexp.MustCompile(`(?i)Revision: 1.10 | ppc`)

				// HACK - Try to guess device type. Works for me ;)
				switch {
				case martemRe.Match([]byte(r.Descr.Value)):
					md := deviceMartem{sd}
					res = &md
				case vType != nil || vErr == nil:
					md := deviceViola{sd}
					res = &md
				case violaRe.Match([]byte(r.Descr.Value)):
					md := deviceViola{sd}
					res = &md
				case violaRe.Match([]byte(bOpts)):
					md := deviceViola{sd}
					res = &md
				case violaHttpRe.Match([]byte(r.Descr.Value)):
					violaWebRe := regexp.MustCompile(`(?ims)<body alink="#3a568d" link="#3a568d" vlink="#3a568d">`)
					body, _ := sd.WebApiGet("")
					if violaWebRe.Match(body) {
						md := deviceViola{sd}
						res = &md
					}
				}
			}
		case d.sysObjectId == ".1.3.6.1.4.1.14988.1":
			md := deviceMikrotik{
				snmpCommon{*d},
			}
			res = &md
		case strings.HasPrefix(d.sysObjectId, ".1.3.6.1.4.1.8691.7."):
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
	} else if strings.HasPrefix(d.sysObjectId, "no-snmp") {
		switch {
		case d.sysObjectId == "no-snmp-ecs":
			res = &deviceEcsEmeter{*d}
		case d.sysObjectId == "no-snmp-viola":
			res = &deviceViolaNoSNMP{*d}
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
	Speed, Last, InOctets, InPkts, InUcast, InMcast, InBcast, InDiscards,
	InErrors, OutOctets, OutPkts, OutUcast, OutMcast, OutBcast, OutDiscards,
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

// Radiolink FarEnd radio interface info
type rlRadioFeIfInfo struct {
	SysName    valString
	Ip         valString
	FeIfDescr  valString
	IfIdx      valInt
	FeIfIdx    valInt
	EntityIdx  valInt
	TxCapacity valInt
	PowerIn    valF64
	PowerOut   valF64
}

// Device sensor value
type sensorVal struct {
	Unit, String string
	Value        uint64
	Divisor      int
	Bool         bool
	IsSet        bool
}

// Onu info
type onuPort struct {
	Id, Speed, Mode valString
	Vlans           []int
	Plugged         valBool
	NativeVlan      valInt
}

type onuInfo struct {
	Model      valString
	Mac        valString
	UpTimeStr  valString
	ConTimeStr valString
	Error      valString
	Version    valString
	OltPort    valString
	Name       valString
	Ports      map[string]onuPort
	TxBytes    sensorVal
	TxPower    sensorVal
	RxPower    sensorVal
	Ram        sensorVal
	Distance   sensorVal
	CpuTemp    sensorVal
	CpuUsage   sensorVal
	DownLimit  sensorVal
	Uplimit    sensorVal
	RxBytes    sensorVal
	ConTime    valU64
	UpTime     valU64
	Online     valBool
	Enabled    valBool
}

// Energy Readings
type eReadings struct {
	day, night sensorVal
	timeStamp  uint
}
