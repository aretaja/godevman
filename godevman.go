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
		case d.sysObjectId == ".1.3.6.1.4.1.28634.14":
			md := deviceComap{
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
				teltonikaRe := regexp.MustCompile(`(?i)teltonika`)
				violaRe := regexp.MustCompile(`(?i)viola`)
				violaHttpRe := regexp.MustCompile(`(?i)Revision: 1.10 | ppc`)

				// HACK - Try to guess device type. Works for me ;)
				switch {
				case martemRe.Match([]byte(r.Descr.Value)):
					md := deviceMartem{sd}
					res = &md
				case teltonikaRe.Match([]byte(r.Descr.Value)):
					md := deviceTeltonika{sd}
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
type ValBool struct {
	Value bool
	IsSet bool
}

type ValF64 struct {
	Value float64
	IsSet bool
}

type ValInt struct {
	Value int
	IsSet bool
}

type ValI64 struct {
	Value int64
	IsSet bool
}

type ValString struct {
	Value string
	IsSet bool
}

type ValU64 struct {
	Value uint64
	IsSet bool
}

// System data
type System struct {
	Descr, ObjectID, Contact, Name, Location, UpTimeStr ValString
	UpTime                                              ValU64
}

// Interface info
type IfInfo struct {
	Descr, Name, Alias, Mac, LastStr, TypeStr, AdminStr, OperStr ValString
	Type, Mtu, Admin, Oper                                       ValI64
	Speed, Last, InOctets, InPkts, InUcast, InMcast, InBcast, InDiscards,
	InErrors, OutOctets, OutPkts, OutUcast, OutMcast, OutBcast, OutDiscards,
	OutErrors ValU64
}

// Interface stack info
type IfStack struct {
	Up, Down map[int][]int
}

// Inventory info
type InvInfo struct {
	Descr, Position, HwProduct, HwRev, Serial, Manufacturer, Model, SwProduct,
	SwRev ValString
	Physical bool
	ParentId ValI64
}

// Dot1Q VLAN bridgeport info
type D1qVlanBrPort struct {
	IfIdx int
	UnTag bool
}

// Dot1Q VLAN info (Boolean in Ports map indicates untagged vlan)
type D1qVlanInfo struct {
	Ports map[int]*D1qVlanBrPort
	Name  string
}

// IP info
type IpInfo struct {
	Mask  string
	IfIdx int64
}

// IP Interface info
type IpIfInfo struct {
	Descr, Alias string
	IpInfo
}

// last backup info
type BackupInfo struct {
	TargetIP, TargetFile string
	Timestamp, Progress  int
	Success              bool
}

// Radiolink radio interface info
type RfInfo struct {
	Name       ValString
	Descr      ValString
	Status     ValString
	Mute       ValBool
	IfIdx      ValInt
	EntityIdx  ValInt
	TxCapacity ValInt
	PowerIn    ValF64
	PowerOut   ValF64
	Snr        ValF64
}

type RauInfo struct {
	Rf        map[string]*RfInfo
	Name      ValString
	Descr     ValString
	EntityIdx ValInt
	Temp      ValF64
}

type RlRadioIfInfo struct {
	Rau       map[string]*RauInfo
	Name      ValString
	Descr     ValString
	AdmStat   ValString
	OperStat  ValString
	IfIdx     ValInt
	EntityIdx ValInt
	Es        ValInt
	Uas       ValInt
}

// Radiolink FarEnd radio interface info
type RlRadioFeIfInfo struct {
	SysName    ValString
	Ip         ValString
	FeIfDescr  ValString
	IfIdx      ValInt
	FeIfIdx    ValInt
	EntityIdx  ValInt
	TxCapacity ValInt
	PowerIn    ValF64
	PowerOut   ValF64
}

// Device sensor value
type SensorVal struct {
	Unit, String string
	Value        uint64
	Divisor      int
	Bool         bool
	IsSet        bool
}

// Onu info
type OnuPort struct {
	Id, Speed, Mode ValString
	Vlans           []int
	Plugged         ValBool
	NativeVlan      ValInt
}

type OnuInfo struct {
	Model      ValString
	Mac        ValString
	UpTimeStr  ValString
	ConTimeStr ValString
	Error      ValString
	Version    ValString
	OltPort    ValString
	Name       ValString
	Ports      map[string]OnuPort
	TxBytes    SensorVal
	TxPower    SensorVal
	RxPower    SensorVal
	Ram        SensorVal
	Distance   SensorVal
	CpuTemp    SensorVal
	CpuUsage   SensorVal
	DownLimit  SensorVal
	Uplimit    SensorVal
	RxBytes    SensorVal
	ConTime    ValU64
	UpTime     ValU64
	Online     ValBool
	Enabled    ValBool
}

// Phase Sync info
type PhaseSyncInfo struct {
	SrcsState     map[string]string
	ParentGmIdent ValString
	ParentGmClass ValString
	State         ValString
	HopsToGm      ValU64
}

// Frequency Sync info
type FreqSyncInfo struct {
	SrcsQaLevel  map[string]string
	ClockMode    ValString
	ClockQaLevel ValString
}

// Mobile modem signal data
type MobSignal struct {
	Registration, Technology, Band, Operator, Ber, CellId, Signal, SignalBars, Imei, Sinr, Rssi, Rsrp, Rsrq SensorVal
}

// Energy Readings
type EReadings struct {
	day, night SensorVal
	timeStamp  uint
}

// Power Generator info
type GenInfo struct {
	GenMode      ValString
	BreakerState ValString
	EngineState  ValString
	GenPower     SensorVal
	RunHours     SensorVal
	BatteryVolt  SensorVal
	GenFreq      SensorVal
	GenCurrentL1 SensorVal
	GenCurrentL2 SensorVal
	GenCurrentL3 SensorVal
	CoolantTemp  SensorVal
	MainsVoltL3  SensorVal
	MainsVoltL2  SensorVal
	GenVoltL2    SensorVal
	FuelConsum   SensorVal
	MainsVoltL1  SensorVal
	FuelLevel    SensorVal
	GenVoltL3    SensorVal
	GenVoltL1    SensorVal
	NumStarts    ValU64
}
