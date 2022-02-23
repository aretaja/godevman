package godevman

import "fmt"

// Adds Ceragon specific SNMP functionality to snmpCommon type
type deviceCeragon struct {
	snmpCommon
}

// Get running software version
func (sd *deviceCeragon) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.2281.10.4.1.13.1.1.4.1"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Execute cli commands
func (sd *deviceCeragon) RunCmds(c []string, o *CliCmdOpts) ([]string, error) {
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
