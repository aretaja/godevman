package godevman

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	"golang.org/x/crypto/ssh"
)

// Adds Ruggedcom specific SNMP functionality to snmpCommon type
type deviceRuggedcom struct {
	snmpCommon
}

// Get running software version
func (sd *deviceRuggedcom) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.15004.4.2.3.3.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Prepare CLI session parameters
func (sd *deviceRuggedcom) cliPrepare() (*CliParams, error) {
	defParams, err := sd.snmpCommon.cliPrepare()
	if err != nil {
		return nil, err
	}

	params := defParams

	// make device specific changes to default parameters
	if sd.cliSession.params.PromptRe == "" {
		params.PromptRe = `>$`
	}
	if sd.cliSession.params.ErrRe == "" {
		params.ErrRe = `(?im)(error|unknown|unrecognized|invalid|not recognized|Examples:|timed out)`
	}
	params.DisconnectCmds = []string{"logout"}
	return params, nil
}

// Create and store cli expect client and update d.cliSession.params
func (sd *deviceRuggedcom) startCli(p *CliParams) error {
	if sd.cliSession.client != nil {
		return nil
	}

	// store sessions parameters
	sd.cliSession.params = p

	// setup connection related vars
	addr := fmt.Sprintf("%s:%s", sd.ip, p.Port)
	if p.Telnet {
		addr = fmt.Sprintf("%s %s", sd.ip, p.Port)
	}

	user := p.Cred[0]
	pass := ""
	if len(p.Cred) > 1 {
		pass = p.Cred[1]
	}

	timeOut := time.Duration(p.Timeout) * time.Second

	verbose := false
	if sd.debug > 0 {
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
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeOut,
	}

	// Regexes for credentials
	uRe := regexp.MustCompile(`Name:\s*$`)
	pRe := regexp.MustCompile(`Password:\s*$`)
	inRe := regexp.MustCompile(`continue...`)
	uiRe := regexp.MustCompile(`X-Logout`)

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

		// Check for continue prompt
		out, _, err := e.Expect(inRe, -1)

		if err != nil {
			return nil, fmt.Errorf("ssh login prompt match failed: %v out: %v", err, out)
		}

		err = e.Send("\r")
		if err != nil {
			return nil, fmt.Errorf("ssh enter failed: %v", err)
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

	// Check for valid ui
	out, _, err := e.Expect(uiRe, -1)

	if err != nil {
		return fmt.Errorf("ssh ui match failed: %v out: %v", err, out)
	}

	err = e.Send("\x13\r")
	if err != nil {
		return fmt.Errorf("ssh ctrl-s failed: %v", err)
	}

	// Check for valid prompt
	re := regexp.MustCompile(p.PromptRe)
	out, _, err = e.Expect(re, -1)
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
	sd.cliSession.client = e

	return nil
}

// Execute cli commands. Returns all sent, received data as string slice
// c - cli commands, f - check for command errors
func (sd *deviceRuggedcom) cliCmds(c []string, f bool) ([]string, error) {
	var output []string
	e := sd.cliSession.client
	if e == nil {
		return output, fmt.Errorf("active cli session not found")
	}

	pRe := regexp.MustCompile(sd.cliSession.params.PromptRe)
	eRe := regexp.MustCompile(sd.cliSession.params.ErrRe)

	cnt := len(c)
	for _, cmd := range c {
		cnt--
		output = append(output, cmd)
		pieces := strings.SplitAfter(cmd, " ")
		last := len(pieces) - 1
		for i, v := range pieces {
			pcmd := v
			if i == last {
				pcmd = pcmd + "\r\n"
			}
			err := e.Send(pcmd)
			if err != nil {
				return output, fmt.Errorf("send(%q) failed: %v", cmd, err)
			}
		}

		// Dont expect specific prompt after last cmd
		if cnt == 0 {
			pRe = regexp.MustCompile(`(?m).*$`)
		}
		out, _, err := e.Expect(pRe, -1)
		out = strings.TrimPrefix(out, cmd+"\r\n")
		output = append(output, out)

		// Check for errors if requested
		if f {
			if eRe.Match([]byte(out)) {
				return output, fmt.Errorf("cli command exec error: %s", out)
			}
		}

		if err != nil {
			return output, fmt.Errorf("expect(%v) failed: %v out: %v", pRe, err, out)
		}
	}

	return output, nil
}

// Execute cli commands
func (sd *deviceRuggedcom) RunCmds(c []string, e bool) ([]string, error) {
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

// Initiate tftp backup of device config
func (sd *deviceRuggedcom) DoBackup() error {
	if sd.backupParams == nil {
		return fmt.Errorf("device backup parameters are not defined")
	}

	// Get backup parameters
	host := sd.backupParams.TargetIp
	if host == "" {
		return fmt.Errorf("target ip is not defined")
	}

	t := time.Now()
	loc, _ := time.LoadLocation(sd.timeZone)
	t = t.In(loc)

	targetFile := sd.backupParams.BasePath + "/" + sd.backupParams.DevIdent + "_" + t.Format(time_iso8601_sec) + ".csv"

	cmds := []string{
		"tftp " + host + " put config.csv " + targetFile,
		"logout",
	}

	res, err := sd.RunCmds(cmds, true)
	if err != nil {
		return fmt.Errorf("cli error: %v, output: %s", err, res)
	}

	return nil
}
