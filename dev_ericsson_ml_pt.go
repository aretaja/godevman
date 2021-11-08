package godevman

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

// Adds Ericsson MINI-LINK PT specific SNMP functionality to snmpCommon type
type deviceEricssonMlPt struct {
	snmpCommon
}

// Get IP Interface info
func (sd *deviceEricssonMlPt) IpIfInfo(ip ...string) (map[string]*ipIfInfo, error) {
	out := make(map[string]*ipIfInfo)

	ipInfo, err := sd.IpInfo([]string{"All"}, ip...)
	if err != nil {
		return out, err
	}

	for i, v := range ipInfo {
		if out[i] == nil {
			out[i] = new(ipIfInfo)
		}
		out[i].ipInfo = *v
	}

	ifInfo, err := sd.Ip6IfDescr()
	if err != nil {
		return out, err
	}

	// Fill output map with interface info
	for i, d := range ipInfo {
		ifIdxStr := strconv.FormatInt(int64(d.IfIdx), 10)
		descr, ok := ifInfo[ifIdxStr]
		if !ok {
			descr = "unkn_" + ifIdxStr
		}

		out[i].Descr = descr
	}

	return out, err
}

// Make http Get request and return byte slice of body.
// Argument string should contain request parameters.
func (sd *deviceEricssonMlPt) WebApiGet(params string) ([]byte, error) {
	client := sd.websession
	if sd.websession == nil {
		// setup client
		c, err := sd.webClient(nil)
		if err != nil {
			return nil, err
		}
		client = c
	}

	res, err := client.Get("https://" + sd.ip + "/cgi-bin/main.fcgi?noCache=" +
		RandomString(13) + "&" + params)
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

// Login via web API and stores web session in deviceEricssonMlPt.websession.
// Use this before use of methods which are accessing restricted device web API.
func (sd *deviceEricssonMlPt) WebAuth(userPass []string) error {
	// setup client
	client, err := sd.webClient(nil)
	if err != nil {
		return err
	}

	// credentials
	cred := url.Values{
		"CATEGORY": {"LOGIN"},
		"USERNAME": {userPass[0]},
		"PASSWORD": {userPass[1]},
	}

	baseUrl := "https://" + sd.ip + "/cgi-bin/main.fcgi?noCache=" + RandomString(13)
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

	var resJson struct {
		Status string
	}

	if err := json.Unmarshal([]byte(body), &resJson); err != nil {
		return err
	}

	if resJson.Status != "Ok" {
		return fmt.Errorf("incorrect username or password")
	}

	// HACK to work around of this issue https://github.com/golang/go/issues/12610
	// Remove ip address from cookie Domain name
	var cookies []*http.Cookie

	urlObj, _ := url.Parse(baseUrl)

	for _, c := range res.Cookies() {
		if net.ParseIP(c.Domain) != nil && c.Domain == urlObj.Host {
			c.Domain = ""
			cookies = append(cookies, c)
		}
	}

	client.Jar.SetCookies(urlObj, cookies)

	sd.websession = client

	return nil
}

// Logout via web API and delete web session from deviceEricssonMlPt.websession.
// Use this after use of methods which are accessing restricted device web API.
func (sd *deviceEricssonMlPt) WebLogout() error {
	if sd.websession == nil {
		return nil
	}

	body, err := sd.WebApiGet("CATEGORY=LOGOUT")
	if err != nil {
		return err
	}

	var resJson struct {
		Status string
	}

	if err := json.Unmarshal([]byte(body), &resJson); err != nil {
		return err
	}

	if resJson.Status != "PENDING" {
		return fmt.Errorf("logout failed: %s", resJson.Status)
	}

	sd.websession = nil

	return nil
}
