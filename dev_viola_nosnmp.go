package godevman

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"

	"github.com/patrickmn/go-cache"
)

// Adds Viola specific functionality to snmpCommon type
type deviceViolaNoSNMP struct {
	device
}

// Make http Get request and return byte slice of body.
// Argument string should contain request parameters.
func (d *deviceViolaNoSNMP) WebApiGet(params string) ([]byte, error) {
	client := d.webSession.client
	if d.webSession.client == nil {
		// setup client
		c, err := d.webClient(nil)
		if err != nil {
			return nil, err
		}
		client = c
	}

	res, err := client.Get("http://" + d.ip + "/" + params)
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
func (sd *deviceViolaNoSNMP) WebApiPostForm(params string, rd url.Values) ([]byte, error) {
	client := sd.webSession.client
	if sd.webSession.client == nil {
		// setup client
		c, err := sd.webClient(nil)
		if err != nil {
			return nil, err
		}
		client = c
	}

	baseUrl := "http://" + sd.ip + "/" + params
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

// Login via web API and stores web session in deviceViolaNoSNMP.webSession.client.
// Use this before use of methods which are accessing restricted device web API.
func (sd *deviceViolaNoSNMP) WebAuth(userPass []string) error {
	// setup client
	client, err := sd.webClient(nil)
	if err != nil {
		return err
	}

	// credentials
	cred := url.Values{
		"FRID": {"0"},
		"user": {userPass[0]},
		"psw":  {userPass[1]},
	}

	baseUrl := "http://" + sd.ip + "/cgi-bin/localconfig"
	// login
	res, err := client.PostForm(baseUrl, cred)
	if err != nil {
		return err
	}

	// close response body
	defer res.Body.Close()

	// read all response body
	body, _ := ioutil.ReadAll(res.Body)
	if res.StatusCode > 299 {
		return fmt.Errorf("response failed with status code: %d", res.StatusCode)
	}

	// HACK device uses obsolete 'META HTTP-EQUIV="Set-Cookie"' to set cookies
	// We need to parse and set cookie manually
	reParts := regexp.MustCompile(`META HTTP-EQUIV="Set-Cookie" CONTENT="(LC_Token)=(\d+);"`)
	parts := reParts.FindStringSubmatch(string(body))

	if len(parts) < 3 {
		return fmt.Errorf("authentication failed")
	}

	c := http.Cookie{
		Name:  parts[1],
		Value: parts[2],
	}

	cookies := []*http.Cookie{&c}
	urlObj, _ := url.Parse(baseUrl)
	client.Jar.SetCookies(urlObj, cookies)

	sd.webSession.client = client

	return nil
}

// Logout via web API and delete web session from deviceEricssonMlPt.webSession.client.
// Use this after use of methods which are accessing restricted device web API.
func (sd *deviceViolaNoSNMP) WebLogout() error {
	if sd.webSession.client == nil {
		return nil
	}

	rd := url.Values{"FRID": {"12000"}}

	_, err := sd.WebApiPostForm("cgi-bin/localconfig", rd)
	if err != nil {
		return fmt.Errorf("errors: WebApiPostForm - %s", err)
	}

	sd.webSession.client = nil

	return nil
}

// Get device model
// Example output:
//  map[string]string{
// 	 "firmware":"IEC-104 RTU 5.2.1 (build 1095)"
// 	 "flash":"8MB"
// 	 "hwserial":"11244151"
// 	 "hwtype":"3.1"
// 	 "mac":"00:06:70:02:72:17"
// 	 "os":"Linux version 2.4.19-uc1"
// 	 "proc":"COLDFIRE(m5272)"
// 	 "prodname":"Arctic Control (EDGE)"
// 	 "ram":"31352 kB"
// 	 "serial":"ACO5272-48-328-027217"
// }
func (sd *deviceViolaNoSNMP) SysInfo() (map[string]string, error) {
	if err := sd.WebAuth(sd.webSession.cred); err != nil {
		return nil, fmt.Errorf("error: WebAuth - %s", err)
	}

	body, err := sd.WebApiGet("cgi-bin/localconfig?1001")
	if err != nil {
		return nil, fmt.Errorf("get request from device api failed: %s", err)
	}

	err = sd.WebLogout()
	if err != nil {
		return nil, fmt.Errorf("errors: WebLogout - %s", err)
	}

	out := make(map[string]string)
	reSep := `.*?</TD>.*?</TD>.*?>\s+`
	reBr := `</TD>.*`

	reParts := regexp.MustCompile(
		"(?s)Product name" + reSep + "(.*?)" + reBr +
			"Product serial number" + reSep + "(.*?)" + reBr +
			"HW serial number" + reSep + "(.*?)" + reBr +
			"HW version" + reSep + "(.*?)" + reBr +
			"Operating system" + reSep + "(.*?)" + reBr +
			"Firmware" + reSep + "(.*?)" + reBr +
			"Processor" + reSep + "(.*?)" + reBr +
			"MAC address" + reSep + "(.*?)" + reBr +
			"RAM memory" + reSep + "(.*?)" + reBr +
			"Flash memory" + reSep + "(.*?)" + reBr)

	parts := reParts.FindStringSubmatch(string(body))
	if len(parts) < 11 {
		return nil, fmt.Errorf("info not found from device web")
	}

	out["prodname"] = parts[1]
	out["serial"] = parts[2]
	out["hwserial"] = parts[3]
	out["hwtype"] = parts[4]
	out["os"] = parts[5]
	out["firmware"] = parts[6]
	out["proc"] = parts[7]
	out["mac"] = parts[8]
	out["ram"] = parts[9]
	out["flash"] = parts[10]

	// save to cache
	sd.cache.Set("SysInfo", out, cache.DefaultExpiration)

	return out, nil
}

// Get info from web
// Valid targets values: "All", "Descr", "ObjectID"
func (sd *deviceViolaNoSNMP) System(targets []string) (system, error) {
	results := func(t []string, i map[string]string) system {
		out := new(system)
		for _, t := range targets {
			if t == "All" || t == "ObjectID" {
				out.ObjectID.Value = "no-snmp-viola"
				out.ObjectID.IsSet = true
			}
			if t == "All" || t == "Descr" {
				out.Descr.Value = i["prodname"]
				out.Descr.IsSet = true
			}
		}

		return *out
	}

	// return from cache if allowed and cache is present
	if sd.useCache {
		if x, ok := sd.cache.Get("SysInfo"); ok {
			info := x.(map[string]string)
			return results(targets, info), nil
		}
	}

	info, err := sd.SysInfo()
	if err != nil {
		return system{}, fmt.Errorf("errors: SysInfo - %s", err)
	}

	return results(targets, info), nil
}

// Get device model
// Example output:
//  map[string]string{
// 	 "hwtype":"3.1",
// 	 "prodname":"Arctic Control (EDGE)",
// 	 "serial":"ACO5272-48-328-027217"
// }
func (sd *deviceViolaNoSNMP) HwInfo() (map[string]string, error) {
	results := func(i map[string]string) map[string]string {
		out := map[string]string{
			"hwtype":   i["hwtype"],
			"prodname": i["prodname"],
			"serial":   i["serial"],
		}
		return out
	}

	// return from cache if allowed and cache is present
	if sd.useCache {
		if x, ok := sd.cache.Get("SysInfo"); ok {
			info := x.(map[string]string)
			return results(info), nil
		}
	}

	info, err := sd.SysInfo()
	if err != nil {
		return nil, fmt.Errorf("errors: SysInfo - %s", err)
	}

	return results(info), nil
}

// Get running software version
func (sd *deviceViolaNoSNMP) SwVersion() (string, error) {

	// return from cache if allowed and cache is present
	if sd.useCache {
		if x, ok := sd.cache.Get("SysInfo"); ok {
			info := x.(map[string]string)
			return info["firmware"], nil
		}
	}

	info, err := sd.SysInfo()
	if err != nil {
		return "", fmt.Errorf("errors: SysInfo - %s", err)
	}

	return info["firmware"], nil
}
