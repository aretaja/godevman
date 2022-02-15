package godevman

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"
)

// Adds Viola specific functionality to snmpCommon type
type deviceViola struct {
	snmpCommon
}

// Make http Get request and return byte slice of body.
// Argument string should contain remainder after base URL.
func (sd *deviceViola) WebApiGet(params string) ([]byte, error) {
	client := sd.webSession.client
	if sd.webSession.client == nil {
		// setup client
		c, err := sd.webClient(nil)
		if err != nil {
			return nil, err
		}
		client = c
	}

	res, err := client.Get("https://" + sd.ip + "/" + params)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode > 299 {
		return body, fmt.Errorf("response failed with status code: %d", res.StatusCode)
	}

	return body, nil
}

// Make http WebApiPostForm request and return byte slice of body.
// Argument string should contain remainder after base URL.
func (sd *deviceViola) WebApiPostForm(params string, rd url.Values) ([]byte, error) {
	client := sd.webSession.client
	if sd.webSession.client == nil {
		// setup client
		c, err := sd.webClient(nil)
		if err != nil {
			return nil, err
		}
		client = c
	}

	baseUrl := "https://" + sd.ip + "/" + params
	res, err := client.PostForm(baseUrl, rd)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode > 299 {
		return body, fmt.Errorf("response failed with status code: %d", res.StatusCode)
	}

	return body, nil
}

// Get PHPSESSID as T url.Values from session coocies
func (sd *deviceViola) getPhpSessid() (url.Values, error) {
	if sd.webSession.client == nil {
		return nil, fmt.Errorf("error: No stored websession")
	}

	client := sd.webSession.client

	urlObj, _ := url.Parse("https://" + sd.ip + "/")
	cookies := client.Jar.Cookies(urlObj)
	if len(cookies) == 0 {
		return nil, fmt.Errorf("error: PHPSESSID not found")
	}

	// request data
	rd := make(url.Values)

	for _, c := range cookies {
		if c.Name == "PHPSESSID" {
			rd[c.Name] = []string{c.Value}
			break
		}
	}

	return rd, nil
}

// Login via web API and stores web session in deviceViola.webSession.client.
// Use this before use of methods which are accessing restricted device web API.
func (sd *deviceViola) WebAuth(userPass []string) error {
	// setup client
	client, err := sd.webClient(nil)
	if err != nil {
		return err
	}

	// credentials
	cred := url.Values{
		"login":    {"Login"},
		"username": {userPass[0]},
		"password": {userPass[1]},
	}

	baseUrl := "https://" + sd.ip + "/index.php"
	// login
	res, err := client.PostForm(baseUrl, cred)
	if err != nil {
		return err
	}

	// close response body
	defer res.Body.Close()

	cookies := res.Cookies()
	if len(cookies) == 0 {
		return fmt.Errorf("authentication failed")
	}

	urlObj, _ := url.Parse(baseUrl)

	client.Jar.SetCookies(urlObj, cookies)

	sd.webSession.client = client

	return nil
}

// Logout via web API and delete web session from deviceEricssonMlPt.webSession.client.
// Use this after use of methods which are accessing restricted device web API.
func (sd *deviceViola) WebLogout() error {
	if sd.webSession.client == nil {
		return nil
	}

	rd, err := sd.getPhpSessid()
	if err != nil {
		return fmt.Errorf("errors: getPhpSessid - %s", err)
	}

	_, err = sd.WebApiPostForm("index.php?logout", rd)
	if err != nil {
		return fmt.Errorf("errors: WebApiPostForm - %s", err)
	}

	sd.webSession.client = nil

	return nil
}

// Get device model
// Example output:
//  map[string]string{
// 	 "hwtype":"0x04",
// 	 "prodname":"Arctic 3G Gateway 2622",
// 	 "serial":"AUG8248-400-328-0257C1"
// }
func (sd *deviceViola) HwInfo() (map[string]string, error) {
	if err := sd.WebAuth(sd.webSession.cred); err != nil {
		return nil, fmt.Errorf("error: WebAuth - %s", err)
	}

	rd, err := sd.getPhpSessid()
	if err != nil {
		return nil, fmt.Errorf("errors: getPhpSessid - %s", err)
	}

	body, err := sd.WebApiPostForm("index.php", rd)
	if err != nil {
		return nil, fmt.Errorf("errors: WebApiPostForm - %s", err)
	}

	err = sd.WebLogout()
	if err != nil {
		return nil, fmt.Errorf("errors: WebLogout - %s", err)
	}

	out := make(map[string]string)
	reParts := regexp.MustCompile(`(?s).*Product name:\s*(.*?)(?:\r|\n).*?Hardware revision:\s*(.*?)(?:\r|\n).*?Device serial number:\s*(.*?)(?:\r|\n)`)
	parts := reParts.FindStringSubmatch(string(body))

	out["prodname"] = parts[1]
	out["hwtype"] = parts[2]
	out["serial"] = parts[3]

	return out, nil
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
