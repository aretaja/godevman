package godevman

import (
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	"golang.org/x/crypto/ssh"
)

// Prepare CLI session parameters
func (d *device) cliPrepare() (*CliParams, error) {
	if d.cliSession == nil {
		return nil, fmt.Errorf("cli parameters missing")
	}

	params := *d.cliSession.params
	if params.PromptRe == "" {
		params.PromptRe = `[>#\$]\s*$`
	}
	if params.ErrRe == "" {
		params.ErrRe = `(?im)(error|unknown|unrecognized|invalid)`
	}
	if params.LineEnd == "" {
		params.LineEnd = "\r\n"
	}
	if params.DisconnectCmds == nil {
		params.DisconnectCmds = []string{"exit"}
	}
	if params.Port == "" {
		params.Port = "22"
		if params.Telnet {
			params.Port = "23"
		}
	}
	if params.Timeout == 0 {
		params.Timeout = 10
	}

	return &params, nil
}

// Create and store cli expect client and update d.cliSession.params
func (d *device) startCli(p *CliParams) error {
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

	var verbose bool
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

	auth := ssh.Password(pass)
	if p.KeyPath != "" {
		key, err := ioutil.ReadFile(p.KeyPath)
		if err != nil {
			log.Printf("warn: unable to read private key: %v", err)
		} else {
			// Create the Signer for this private key.
			var signer ssh.Signer
			if p.KeySecret != "" {
				signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(p.KeySecret))
				if err != nil {
					log.Printf("warn: unable to parse password protected private key: %v", err)
				}
			} else {
				signer, err = ssh.ParsePrivateKey(key)
				if err != nil {
					log.Printf("warn: unable to parse private key: %v", err)
				}
			}
			auth = ssh.PublicKeys(signer)
		}
	}

	cconf := &ssh.ClientConfig{
		Config:          config,
		User:            user,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeOut,
	}

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
		return e, nil
	}

	telnetExpecter := func() (*expect.GExpect, error) {
		e, _, err := expect.Spawn(fmt.Sprintf("telnet %s", addr), timeOut, expect.Verbose(verbose))
		if err != nil {
			return nil, fmt.Errorf("create telnet expecter failed: %v", err)
		}

		// Check for valid login prompt
		uRe := regexp.MustCompile(`(?i)(ogin:|name:|as:)\s*$`)
		out, _, err := e.Expect(uRe, -1)

		if err != nil {
			return nil, fmt.Errorf("telnet login prompt match failed: %v out: %v", err, out)
		}

		err = e.Send(user + p.LineEnd)
		if err != nil {
			return nil, fmt.Errorf("telnet send username failed: %v", err)
		}

		// Check for valid password prompt
		pRe := regexp.MustCompile(`(?i)pass.*:\s*$`)
		out, _, err = e.Expect(pRe, -1)
		if err != nil {
			return nil, fmt.Errorf("telnet password prompt match failed: %v out: %v", err, out)
		}

		err = e.Send(pass + p.LineEnd)
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
		err := e.Send(cmd + p.LineEnd)
		if err != nil {
			return fmt.Errorf("send(%q) failed: %v", cmd, err)
		}

		out, _, err := e.Expect(re, -1)
		out = strings.TrimPrefix(out, cmd+p.LineEnd)
		if err != nil {
			return fmt.Errorf("expect(%v) failed: %v out: %v", re, err, out)
		}
	}

	// Store cli client and parameters
	d.cliSession.client = e

	return nil
}

// Execute cli commands. Returns all sent, received data as string slice
// c - cli commands, e - check for command errors
func (d *device) cliCmds(c []string, f bool) ([]string, error) {
	var output []string
	e := d.cliSession.client
	if e == nil {
		return output, fmt.Errorf("active cli session not found")
	}

	pRe := regexp.MustCompile(d.cliSession.params.PromptRe)
	eRe := regexp.MustCompile(d.cliSession.params.ErrRe)

	cnt := len(c)
	for _, cmd := range c {
		cnt--
		output = append(output, cmd)
		err := e.Send(cmd + d.cliSession.params.LineEnd)
		if err != nil {
			return output, fmt.Errorf("send(%q) failed: %v", cmd, err)
		}

		// Dont expect specific prompt after last cmd
		if cnt == 0 {
			pRe = regexp.MustCompile(`(?m).*$`)
		}
		out, _, err := e.Expect(pRe, -1)
		out = strings.TrimPrefix(out, cmd+d.cliSession.params.LineEnd)
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

// Close cli expect client
func (d *device) closeCli() error {
	e := d.cliSession.client
	if e == nil {
		return nil
	}

	if d.cliSession.params.DisconnectCmds != nil {
		for _, cmd := range d.cliSession.params.DisconnectCmds {
			err := e.Send(cmd + d.cliSession.params.LineEnd)
			if err != nil {
				break
			}
			time.Sleep(time.Second)
		}
	}
	d.cliSession.client = nil
	return nil
}
