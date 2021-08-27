// Package godevman - common godevman package
package godevman

import (
	"fmt"
	"net"
	"os"
	"regexp"

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
			return nil, fmt.Errorf("sysobjectid and sysname discovery failed - snmp error: %v", err)
		}

		d.sysname = res[oids["sysname"]].OctetString
		if val, ok := oids["sysobjectid"]; ok {
			d.sysobjectid = res[val].ObjectIdentifier
		}
	}

	// DEBUG
	if _, set := os.LookupEnv("GODEVMAN_DEBUG"); set {
		fmt.Printf("%# v\n", pretty.Formatter(d))
	}

	return &d, nil
}

// Morph - Type morphing according to device
func (d *device) Morph() interface{} {
	var res interface{} = d
	switch {
	case d.sysobjectid == ".1.3.6.1.4.1.14988.1":
		md := deviceMikrotik{
			snmpCommon{*d},
		}
		res = &md
	}

	return res
}
