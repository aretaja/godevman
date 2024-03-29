package godevman

import (
	"fmt"
	"strings"
)

// Adds Linux specific SNMP functionality to snmpCommon type
type deviceLinux struct {
	snmpCommon
}

// Get running software version
func (sd *deviceLinux) SwVersion() (string, error) {
	// find index for kernel uname if any (must be configured in device snmpd.conf)
	oid := ".1.3.6.1.4.1.8072.1.3.2.2.1.2"
	r, err := sd.snmpSession.Walk(oid, true, true)
	if err != nil && sd.handleErr(err) {
		return "", err
	}

	var token string
	for k, v := range r {
		if strings.HasSuffix(v.OctetString, "/uname") {
			token = k
			break
		}
	}

	version := "Na"
	if len(token) != 0 {
		oid = ".1.3.6.1.4.1.8072.1.3.2.3.1.4." + token
		r, err = sd.getone(oid)
		if err != nil && sd.handleErr(err) {
			return version, err
		}

		if r[oid].Vtype == "Integer" && r[oid].Integer == 0 {
			oid = ".1.3.6.1.4.1.8072.1.3.2.3.1.1." + token
			r, err = sd.getone(oid)
			if err != nil && sd.handleErr(err) {
				return version, err
			}

			version = r[oid].OctetString
		}
	}

	return version, err
}

// Options passed to the configure script when this agent was built
func (sd *deviceLinux) BuildOpts() (string, error) {
	oid := ".1.3.6.1.4.1.2021.100.6.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Execute cli commands
func (sd *deviceLinux) RunCmds(c []string, o *CliCmdOpts) ([]string, error) {
	if o == nil {
		o = new(CliCmdOpts)
	}

	p, err := sd.cliPrepare()
	if err != nil {
		return nil, err
	}

	err = sd.startCli(p)
	if err != nil {
		return nil, err
	}

	out, err := sd.cliCmds(c, o.ChkErr)
	if err != nil {
		err2 := sd.closeCli()
		if err2 != nil {
			err = fmt.Errorf("%v; session close error: %v", err, err2)
		}
		return out, err
	}

	err = sd.closeCli()
	if err != nil {
		return out, err
	}

	return out, nil
}
