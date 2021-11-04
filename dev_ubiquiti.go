package godevman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Adds Ubiquiti specific SNMP functionality to snmpCommon type
type deviceUbiquiti struct {
	snmpCommon
}

// Get running software version
func (sd *deviceUbiquiti) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.41112.1.5.1.3.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Login via web API and stores web session in deviceUbiquiti.websession.
// Use this before use of methods which are accessing restricted device web API.
func (sd *deviceUbiquiti) WebAuth(userPass []string) error {
	// setup client
	client, err := sd.webClient(nil)
	if err != nil {
		return err
	}

	baseUrl := "https://" + sd.ip + "/api/v1.0/user/login"
	values := map[string]string{"username": userPass[0], "password": userPass[1]}

	json_data, err := json.Marshal(values)
	if err != nil {
		return err
	}

	// login
	res, err := client.Post(baseUrl, "application/json", bytes.NewBuffer(json_data))
	if err != nil {
		return err
	}

	// close response body
	defer res.Body.Close()

	if res.StatusCode > 299 {
		return fmt.Errorf("response failed with status code: %d", res.StatusCode)
	}

	token := res.Header.Values("X-Auth-Token")
	if token != nil {
		client, err = sd.webClient(map[string][]string{"X-Auth-Token": token})
		if err != nil {
			return err
		}
	}

	sd.websession = client

	return nil
}

// Logout via web API and delete web session from deviceUbiquiti.websession.
// Use this after use of methods which are accessing restricted device web API.
func (sd *deviceUbiquiti) WebLogout() error {
	if sd.websession == nil {
		return nil
	}

	res, err := sd.websession.Post("https://"+sd.ip+"/api/v1.0/user/logout", "application/json", nil)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)
	if res.StatusCode > 299 {
		return fmt.Errorf("response failed with status code: %d", res.StatusCode)
	}

	var resJson struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}

	if err := json.Unmarshal([]byte(body), &resJson); err != nil {
		return err
	}

	if resJson.Message != "Success" {
		msg := "web API logout failed"
		if resJson.Detail != "" {
			msg += " - " + resJson.Detail
		}
		return fmt.Errorf(msg)
	}

	sd.websession = nil

	return nil
}
