package godevman

import (
	"fmt"
	"regexp"
	"strings"
)

// Adds Viola specific functionality to snmpCommon type
type deviceViola struct {
	snmpCommon
}

// Prepare CLI session parameters
func (sd *deviceViola) cliPrepare() (*CliParams, error) {
	defParams, err := sd.snmpCommon.cliPrepare()
	if err != nil {
		return nil, err
	}

	params := defParams

	// make device specific changes to default parameters
	if sd.cliSession.params.LineEnd == "" {
		params.LineEnd = "\n"
	}

	return params, nil
}

// Execute cli commands
func (sd *deviceViola) RunCmds(c []string, o *CliCmdOpts) ([]string, error) {
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

	if o.Priv {
		err = sd.cliPrivileged()
		if err != nil {
			return nil, err
		}
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

// Get privileged mode (su -)
func (d *device) cliPrivileged() error {
	e := d.cliSession.client
	if e == nil {
		return fmt.Errorf("active cli session not found")
	}

	p := d.cliSession.params

	if len(p.Cred) < 3 {
		return fmt.Errorf("privileged user credentials not found")
	}

	pass := p.Cred[2]
	pRe := regexp.MustCompile(p.PromptRe)
	passRe := regexp.MustCompile(`(?i)password: *$`)
	eRe := regexp.MustCompile(p.ErrRe)

	err := e.Send("su -" + p.LineEnd)
	if err != nil {
		return fmt.Errorf("send su command failed: %v", err)
	}
	out, _, err := e.Expect(passRe, -1)
	if err != nil {
		return fmt.Errorf("cli privileged password prompt mismatch: %s", out)
	}

	err = e.Send(pass + p.LineEnd)
	if err != nil {
		return fmt.Errorf("send privileged user password failed: %v", err)
	}
	out, _, err = e.Expect(pRe, -1)
	out = strings.TrimPrefix(out, pass+p.LineEnd)
	if err != nil {
		return fmt.Errorf("cli prompt mismatch: %s", out)
	}
	// Check for errors
	if eRe.Match([]byte(out)) {
		return fmt.Errorf("cli privileged user password error: %s", out)
	}

	return nil
}

// Get running software version
func (sd *deviceViola) SwVersion() (string, error) {
	cmds := []string{
		"firmware -v",
		"exit",
	}

	r, err := sd.RunCmds(cmds, &CliCmdOpts{ChkErr: true})
	if err != nil {
		return "", fmt.Errorf("cli command error: %v", err)
	}

	var rows []string
	for _, s := range r {
		rows = append(rows, SplitLineEnd(s)...)
	}

	return rows[1], nil
}

// Get device model
// Example output:
//  map[string]string{
// 	 "hwtype":"0x04",
// 	 "prodname":"Arctic 3G Gateway 2622",
// 	 "serial#":"AUG8248-400-328-0257C1"
// }
func (sd *deviceViola) HwInfo() (map[string]string, error) {
	cmds := []string{
		"fw_printenv prodname hwtype serial#",
		"exit",
		"exit",
	}

	r, err := sd.RunCmds(cmds, &CliCmdOpts{ChkErr: true, Priv: true})
	if err != nil {
		return nil, fmt.Errorf("cli command error: %v", err)
	}

	out := make(map[string]string)

	var rows []string
	for _, s := range r {
		rows = append(rows, SplitLineEnd(s)...)
	}

	for _, p := range rows {
		if !strings.Contains(p, "=") {
			continue
		}

		param := strings.Split(p, "=")
		out[param[0]] = param[1]
	}

	return out, nil
}
