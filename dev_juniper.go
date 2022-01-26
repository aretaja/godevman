package godevman

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
	if params.PromptRe == "" {
		params.PromptRe = `(>|#|\%) ?$`
	}
	if params.ErrRe == "" {
		params.ErrRe = `(?im)(error|unknown|invalid|failed|timed out)`
	}
	if params.PreCmds == nil {
		params.PreCmds = []string{
			"",
			"set cli complete-on-space off",
			"set cli screen-length 0",
		}
	}

	return params, nil
}

// Execute cli commands
func (sd *deviceJuniper) RunCmds(c []string) ([]string, error) {
	p, err := sd.cliPrepare()
	if err != nil {
		return nil, err
	}

	err = sd.startCli(p)
	if err != nil {
		return nil, err
	}

	out, err := sd.cliCmds(c)
	if err != nil {
		return out, err
	}

	err = sd.closeCli()
	if err != nil {
		return out, err
	}

	return out, nil
}
