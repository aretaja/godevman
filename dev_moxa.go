package godevman

import (
	"fmt"
	"regexp"
)

// Adds Moxa specific SNMP functionality to snmpCommon type
type deviceMoxa struct {
	snmpCommon
}

// Get running software version
func (sd *deviceMoxa) SwVersion() (string, error) {
	oid := sd.sysObjectId + ".1.4.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Prepare CLI session parameters
func (sd *deviceMoxa) cliPrepare() (*CliParams, error) {
	defParams, err := sd.snmpCommon.cliPrepare()
	if err != nil {
		return nil, err
	}

	params := defParams

	// make device specific changes to default parameters
	if params.DisconnectCmds == nil {
		params.DisconnectCmds = []string{"end", "exit"}
	}
	if params.PreCmds == nil {
		params.PreCmds = []string{
			"terminal length 0",
		}
	}

	return params, nil
}

// Execute cli commands
func (sd *deviceMoxa) RunCmds(c []string, e bool) ([]string, error) {
	p, err := sd.cliPrepare()
	if err != nil {
		return nil, err
	}

	err = sd.startCli(p)
	if err != nil {
		return nil, err
	}

	out, err := sd.cliCmds(c, e)
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

// Get running config
func (sd *deviceMoxa) RuningCfg() (string, error) {
	cmds := []string{"sho run", "exit"}
	res, err := sd.RunCmds(cmds, true)
	if err != nil {
		return "", fmt.Errorf("cli command error: %v", err)
	}

	if len(res) < 2 {
		return "", fmt.Errorf("'sho run' output has no data to capture")
	}

	re := regexp.MustCompile(`(?is)^.*?(!.*\n).*$`)
	m := re.FindStringSubmatch(res[1])

	if len(m) < 2 {
		return "", fmt.Errorf("can't find config from 'sho run' output")
	}

	return m[1], nil
}
