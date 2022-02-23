package godevman

import (
	"fmt"
)

// Adds Juniper specific SNMP functionality to snmpCommon type
type deviceJuniper struct {
	snmpCommon
}

// Get running software version
func (sd *deviceJuniper) SwVersion() (string, error) {
	oid := ".1.3.6.1.2.1.25.6.3.1.2.2"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Prepare CLI session parameters
func (sd *deviceJuniper) cliPrepare() (*CliParams, error) {
	defParams, err := sd.snmpCommon.cliPrepare()
	if err != nil {
		return nil, err
	}

	params := defParams

	// make device specific changes to default parameters
	params.Timeout = 30
	if sd.cliSession.params.PromptRe == "" {
		params.PromptRe = `(>|#|\%) ?$`
	}
	if sd.cliSession.params.ErrRe == "" {
		params.ErrRe = `(?im)(error|unknown|invalid|failed|timed out)`
	}
	if sd.cliSession.params.PreCmds == nil {
		params.PreCmds = []string{
			"",
			"set cli complete-on-space off",
			"set cli screen-length 0",
		}
	}

	return params, nil
}

// Execute cli commands
func (sd *deviceJuniper) RunCmds(c []string, o *CliCmdOpts) ([]string, error) {
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

// Set via CLI
// Set Interface Alias
// set - map of ifIndexes and related ifAliases
func (sd *deviceJuniper) SetIfAlias(set map[string]string) error {
	idxs := make([]string, 0, len(set))
	for k := range set {
		idxs = append(idxs, k)
	}

	r, err := sd.IfInfo([]string{"Descr", "Alias"}, idxs...)
	if err != nil {
		return fmt.Errorf("ifinfo error: %v", err)
	}

	cmds := []string{"configure"}

	for k, v := range set {
		if i, ok := r[k]; ok {
			if i.Alias.IsSet && i.Alias.Value != v && i.Descr.IsSet {
				cmd := "set interfaces " + i.Descr.Value + " description " + v
				cmds = append(cmds, cmd)
			}
		}
	}

	cmds = append(cmds, "commit and-quit", "exit")

	_, err = sd.RunCmds(cmds, &CliCmdOpts{ChkErr: true})
	if err != nil {
		return fmt.Errorf("cli command error: %v", err)
	}

	return nil
}
