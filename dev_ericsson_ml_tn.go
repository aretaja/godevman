package godevman

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	"golang.org/x/crypto/ssh"
)

// Adds Ericsson MINI-LINK TN specific SNMP functionality to snmpCommon type
type deviceEricssonMlTn struct {
	snmpCommon
}

// Get running software version
func (sd *deviceEricssonMlTn) SwVersion() (string, error) {
	if sd.sysObjectId == ".1.3.6.1.4.1.193.81.1.1.1" { // Compact Node
		oid := ".1.3.6.1.4.1.193.81.2.7.1.1.1.4.1.1"
		r, err := sd.getone(oid)
		return strings.TrimSpace(r[oid].OctetString), err
	}

	// get installed sw states
	oid := ".1.3.6.1.4.1.193.81.2.7.1.3.0"
	r, err := sd.getone(oid)
	if err != nil && sd.handleErr(err) {
		return "", err
	}

	oid = ".1.3.6.1.4.1.193.81.2.7.1.2.1.3." + fmt.Sprintf("%d", r[oid].Integer)
	r, err = sd.getone(oid)
	if err != nil && sd.handleErr(err) {
		return "", err
	}

	return strings.TrimSpace(r[oid].OctetString), err
}

// Prepare CLI session parameters
func (sd *deviceEricssonMlTn) cliPrepare() (*CliParams, error) {
	defParams, err := sd.snmpCommon.cliPrepare()
	if err != nil {
		return nil, err
	}

	params := defParams

	// make device specific changes to default parameters
	if sd.cliSession.params.PromptRe == "" {
		params.PromptRe = `(>|#)\s?$`
	}
	if sd.cliSession.params.ErrRe == "" {
		params.ErrRe = `(?im)(error|unknown|invalid|failed|timed out)`
	}
	if sd.cliSession.params.DisconnectCmds == nil {
		params.DisconnectCmds = []string{"end", "exit"}
	}

	return params, nil
}

// Create and store cli expect client and update d.cliSession.params
func (d *deviceEricssonMlTn) startCli(p *CliParams) error {
	if d.cliSession.client != nil {
		return nil
	}

	// store sessions parameters
	d.cliSession.params = p

	// setup connection related vars
	addr := fmt.Sprintf("%s:%s", d.ip, p.Port)
	if p.Telnet {
		addr = fmt.Sprintf("%s %s", d.ip, p.Port)
	}

	user := p.Cred[0]
	pass := ""
	if len(p.Cred) > 1 {
		pass = p.Cred[1]
	}

	timeOut := time.Duration(p.Timeout) * time.Second

	// verbose := false
	verbose := false
	if d.debug > 0 {
		verbose = true
	}

	// Allow weaker key exchange algorithms
	var config ssh.Config
	config.SetDefaults()
	kexOrder := config.KeyExchanges
	kexOrder = append(kexOrder, "diffie-hellman-group1-sha1", "diffie-hellman-group-exchange-sha1", "diffie-hellman-group-exchange-sha256")
	config.KeyExchanges = kexOrder

	ciOrder := config.Ciphers
	ciOrder = append(ciOrder, "aes256-cbc", "aes192-cbc", "aes128-cbc", "3des-cbc")
	config.Ciphers = ciOrder

	cconf := &ssh.ClientConfig{
		Config:          config,
		User:            "cli",
		Auth:            []ssh.AuthMethod{ssh.Password("")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeOut,
	}

	// Regexes for credentials
	uRe := regexp.MustCompile(`User: ?$`)
	pRe := regexp.MustCompile(`Password: ?$`)

	// Create expecter
	sshExpecter := func() (*expect.GExpect, error) {
		sshClt, err := ssh.Dial("tcp", addr, cconf)
		if err != nil {
			return nil, fmt.Errorf("ssh connection to %s failed: %v", addr, err)
		}

		e, _, err := expect.SpawnSSH(sshClt, timeOut, expect.Verbose(verbose))
		if err != nil {
			return nil, fmt.Errorf("create ssh expecter failed: %v", err)
		}

		// Check for valid login prompt
		out, _, err := e.Expect(uRe, -1)

		if err != nil {
			return nil, fmt.Errorf("ssh login prompt match failed: %v out: %v", err, out)
		}

		err = e.Send(user + "\r")
		if err != nil {
			return nil, fmt.Errorf("ssh send username failed: %v", err)
		}

		// Check for valid password prompt
		out, _, err = e.Expect(pRe, -1)
		if err != nil {
			return nil, fmt.Errorf("ssh password prompt match failed: %v out: %v", err, out)
		}

		err = e.Send(pass + "\r")
		if err != nil {
			return nil, fmt.Errorf("ssh send password failed: %v", err)
		}

		return e, nil
	}

	telnetExpecter := func() (*expect.GExpect, error) {
		e, _, err := expect.Spawn(fmt.Sprintf("telnet %s", addr), timeOut, expect.Verbose(verbose))
		if err != nil {
			return nil, fmt.Errorf("create telnet expecter failed: %v", err)
		}

		// Check for valid login prompt
		out, _, err := e.Expect(uRe, -1)

		if err != nil {
			return nil, fmt.Errorf("telnet login prompt match failed: %v out: %v", err, out)
		}

		err = e.Send(user + "\r")
		if err != nil {
			return nil, fmt.Errorf("telnet send username failed: %v", err)
		}

		// Check for valid password prompt
		out, _, err = e.Expect(pRe, -1)
		if err != nil {
			return nil, fmt.Errorf("telnet password prompt match failed: %v out: %v", err, out)
		}

		err = e.Send(pass + "\r")
		if err != nil {
			return nil, fmt.Errorf("telnet send password failed: %v", err)
		}
		return e, nil
	}

	e := new(expect.GExpect)
	switch p.Telnet {
	case false:
		// ssh session
		s, err := sshExpecter()
		if err != nil {
			return err
		}
		e = s
	case true:
		// telnet session (requires local telnet client)
		s, err := telnetExpecter()
		if err != nil {
			return err
		}
		e = s
	}

	// Check for valid prompt
	re := regexp.MustCompile(p.PromptRe)
	out, _, err := e.Expect(re, -1)
	if err != nil {
		return fmt.Errorf("prompt(%v) match failed: %v out: %v", re, err, out)
	}

	// Run Initial commands if requested
	for _, cmd := range p.PreCmds {
		err := e.Send(cmd + "\r")
		if err != nil {
			return fmt.Errorf("send(%q) failed: %v", cmd, err)
		}

		out, _, err := e.Expect(re, -1)
		out = strings.TrimPrefix(out, cmd+"\r")
		if err != nil {
			return fmt.Errorf("expect(%v) failed: %v out: %v", re, err, out)
		}
	}

	// Store cli client and parameters
	d.cliSession.client = e

	return nil
}

// Execute cli commands
func (sd *deviceEricssonMlTn) RunCmds(c []string, o *CliCmdOpts) ([]string, error) {
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
