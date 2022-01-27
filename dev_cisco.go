package godevman

import (
	"fmt"
	"regexp"
)

// Adds Cisco specific SNMP functionality to snmpCommon type
type deviceCisco struct {
	snmpCommon
}

// Get running software version
func (sd *deviceCisco) SwVersion() (string, error) {
	var out string
	res, err := sd.System([]string{"Descr"})
	if err != nil {
		return out, err
	}

	re := regexp.MustCompile(`Version (.*?)[,|\s]`)
	if res.Descr.IsSet {
		reMatch := re.FindStringSubmatch(res.Descr.Value)
		if reMatch == nil {
			return out, fmt.Errorf("failed to parse sysDescr for version - %s", res.Descr.Value)
		}
		out = reMatch[1]
	}

	return out, nil
}

// Prepare CLI session parameters
func (sd *deviceCisco) cliPrepare() (*CliParams, error) {
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
			"terminal width 132",
		}
	}

	return params, nil
}

// Execute cli commands
func (sd *deviceCisco) RunCmds(c []string, e bool) ([]string, error) {
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
