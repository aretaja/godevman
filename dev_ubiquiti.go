package godevman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/aretaja/snmphelper"
	"github.com/patrickmn/go-cache"
)

// Adds Ubiquiti specific SNMP functionality to snmpCommon type
type deviceUbiquiti struct {
	snmpCommon
}

// Ubiquiti specific OLT interface info type used by device web API
type UbiOltInterfaceIdentification struct {
	ID   *string `json:"id"`
	Mac  *string `json:"mac"`
	Name *string `json:"name"`
	Type *string `json:"type"`
}

type UbiOltInterfaceSfp struct {
	Sfp struct {
		Los     interface{} `json:"los"`
		Serial  *string     `json:"serial"`
		TxFault interface{} `json:"txFault"`
		Part    *string     `json:"part"`
		Vendor  *string     `json:"vendor"`
		Present *bool       `json:"present"`
	}
}

type UbiOltInterfaceStatus struct {
	CurrentSpeed *string `json:"currentSpeed"`
	Speed        *string `json:"speed,omitempty"`
	Enabled      *bool   `json:"enabled"`
	Plugged      *bool   `json:"plugged"`
}

type UbiOltInterfaceLag struct {
	Interfaces []interface{} `json:"interfaces"`
	Static     *bool         `json:"static"`
}

type UbiOltInterface struct {
	Addresses      []interface{}                 `json:"addresses"`
	Identification UbiOltInterfaceIdentification `json:"identification,omitempty"`
	Pon            UbiOltInterfaceSfp            `json:"pon,omitempty"`
	Status         UbiOltInterfaceStatus         `json:"status,omitempty"`
	Port           UbiOltInterfaceSfp            `json:"port,omitempty"`
	Lag            UbiOltInterfaceLag            `json:"lag,omitempty"`
}

type UbiOltInterfaces []UbiOltInterface

type UbiOltInterfacePonSet struct {
	Addresses      []interface{}                 `json:"addresses"`
	Identification UbiOltInterfaceIdentification `json:"identification,omitempty"`
	Status         UbiOltInterfaceStatus         `json:"status,omitempty"`
	Pon            UbiOltInterfaceSfp            `json:"pon,omitempty"`
}

type UbiOltInterfacePortSet struct {
	Addresses      []interface{}                 `json:"addresses"`
	Identification UbiOltInterfaceIdentification `json:"identification,omitempty"`
	Status         UbiOltInterfaceStatus         `json:"status,omitempty"`
	Port           UbiOltInterfaceSfp            `json:"port,omitempty"`
}

// Ubiquiti specific OLT statistics type used by device web API
type UbiOltStatistics []struct {
	Interfaces []struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Statistics struct {
			RxBroadcast uint64 `json:"rxBroadcast"`
			RxBytes     uint64 `json:"rxBytes"`
			RxErrors    uint64 `json:"rxErrors"`
			RxMulticast uint64 `json:"rxMulticast"`
			RxPackets   uint64 `json:"rxPackets"`
			RxRate      uint64 `json:"rxRate"`
			TxBroadcast uint64 `json:"txBroadcast"`
			TxBytes     uint64 `json:"txBytes"`
			TxErrors    uint64 `json:"txErrors"`
			TxMulticast uint64 `json:"txMulticast"`
			TxPackets   uint64 `json:"txPackets"`
			TxRate      uint64 `json:"txRate"`
		} `json:"statistics,omitempty"`
	} `json:"interfaces"`
	Device struct {
		CPU []struct {
			Identifier  string  `json:"identifier"`
			Temperature float64 `json:"temperature"`
			Usage       int     `json:"usage"`
		} `json:"cpu"`
		FanSpeeds []struct {
			Value float64 `json:"value"`
		} `json:"fanSpeeds"`
		Power []struct {
			PsuType   string  `json:"psuType"`
			Current   float64 `json:"current,omitempty"`
			Power     float64 `json:"power,omitempty"`
			Voltage   float64 `json:"voltage,omitempty"`
			Connected bool    `json:"connected"`
		} `json:"power"`
		Signals []interface{} `json:"signals"`
		Storage []struct {
			Name        string  `json:"name"`
			SysName     string  `json:"sysName"`
			Type        string  `json:"type"`
			Size        int     `json:"size"`
			Temperature float64 `json:"temperature"`
			Used        int     `json:"used"`
		} `json:"storage"`
		Temperatures []struct {
			Value float64 `json:"value"`
		} `json:"temperatures"`
		RAM struct {
			Free  int `json:"free"`
			Total int `json:"total"`
			Usage int `json:"usage"`
		} `json:"ram"`
		Uptime int `json:"uptime"`
	} `json:"device"`
	Timestamp int64 `json:"timestamp"`
}

// Get running software version
func (sd *deviceUbiquiti) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.41112.1.5.1.3.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Get ifNumber
func (sd *deviceUbiquiti) IfNumber() (int64, error) {
	var out int64
	oid := ".1.3.6.1.4.1.41112.1.5.7.2.1.1"
	r, err := sd.snmpSession.Walk(oid, true, true)
	if err != nil && sd.handleErr(oid, err) {
		return out, err
	}

	out = int64(len(r))

	return out, nil
}

// Make http GET request and return byte slice of body.
// Argument string should contain request parameters.
func (sd *deviceUbiquiti) WebApiGet(params string) ([]byte, error) {
	client := sd.webSession.client
	if sd.webSession.client == nil {
		// setup client
		c, err := sd.webClient(nil)
		if err != nil {
			return nil, err
		}
		client = c
	}

	res, err := client.Get("https://" + sd.ip + "/api/v1.0/" + params)
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

// Make http POST request and return byte slice of body.
// Argument string should contain request parameters.
func (sd *deviceUbiquiti) WebApiPost(target string, jsonData []byte) ([]byte, error) {
	client := sd.webSession.client
	if sd.webSession.client == nil {
		// setup client
		c, err := sd.webClient(nil)
		if err != nil {
			return nil, err
		}
		client = c
	}

	baseUrl := "https://" + sd.ip + "/api/v1.0/"
	res, err := client.Post(baseUrl+target, "application/json", bytes.NewBuffer(jsonData))
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

// Make http PUT request and return byte slice of body.
// Argument string should contain request parameters.
func (sd *deviceUbiquiti) WebApiPut(target string, jsonData []byte) ([]byte, error) {
	client := sd.webSession.client
	if sd.webSession.client == nil {
		// setup client
		c, err := sd.webClient(nil)
		if err != nil {
			return nil, err
		}
		client = c
	}

	baseUrl := "https://" + sd.ip + "/api/v1.0/"
	req, err := http.NewRequest(http.MethodPut, baseUrl+target, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res, err := client.Do(req)
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

	jsonData, err := json.Marshal(values)
	if err != nil {
		return err
	}

	// login
	res, err := client.Post(baseUrl, "application/json", bytes.NewBuffer(jsonData))
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

	sd.webSession.client = client

	return nil
}

// Logout via web API and delete web session from deviceUbiquiti.websession.
// Use this after use of methods which are accessing restricted device web API.
func (sd *deviceUbiquiti) WebLogout() error {
	if sd.webSession == nil {
		return nil
	}

	res, err := sd.webSession.client.Post("https://"+sd.ip+"/api/v1.0/user/logout", "application/json", nil)
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
	sd.webSession.client = nil

	return nil
}

// Get all OLT interface info via web API.
func (sd *deviceUbiquiti) oltIfInfo() (*UbiOltInterfaces, error) {
	// return from cache if allowed and cache is present
	if sd.useCache {
		if x, found := sd.cache.Get("oltIfInfo"); found {
			return x.(*UbiOltInterfaces), nil
		}
	}

	if err := sd.WebAuth(sd.webSession.cred); err != nil {
		return nil, fmt.Errorf("error: WebAuth - %s", err)
	}

	body, err := sd.WebApiGet("interfaces")
	if err != nil {
		return nil, fmt.Errorf("get request from device api failed: %s", err)
	}

	err = sd.WebLogout()
	if err != nil {
		return nil, fmt.Errorf("errors: WebLogout - %s", err)
	}

	info := new(UbiOltInterfaces)
	err = json.Unmarshal(body, info)
	if err != nil {
		return nil, fmt.Errorf("unmarshal OLT interface info failed: %s", err)
	}

	if info == nil {
		return nil, fmt.Errorf("no OLT interface info")
	}

	// save to cache
	sd.cache.Set("oltIfInfo", info, cache.DefaultExpiration)

	return info, nil
}

// Get all OLT statistics via web API.
func (sd *deviceUbiquiti) oltStatistics() (*UbiOltStatistics, error) {
	// return from cache if allowed and cache is present
	if sd.useCache {
		if x, found := sd.cache.Get("oltStatistics"); found {
			return x.(*UbiOltStatistics), nil
		}
	}

	if err := sd.WebAuth(sd.webSession.cred); err != nil {
		return nil, fmt.Errorf("error: WebAuth - %s", err)
	}

	body, err := sd.WebApiGet("statistics")
	if err != nil {
		return nil, fmt.Errorf("get request from device api failed: %s", err)
	}

	err = sd.WebLogout()
	if err != nil {
		return nil, fmt.Errorf("errors: WebLogout - %s", err)
	}

	info := new(UbiOltStatistics)
	err = json.Unmarshal(body, info)
	if err != nil {
		return nil, fmt.Errorf("unmarshal OLT statistics failed: %s", err)
	}

	if info == nil {
		return nil, fmt.Errorf("no OLT statistics")
	}

	// save to cache
	sd.cache.Set("oltStatistics", info, cache.DefaultExpiration)

	return info, nil
}

// Get info from .iso.org.dod.internet.private.enterprises.ubnt.ubntMIB.ubntEdgeMax.ubntSfps.ubntSfpsTable and device web API
// Valid targets values: "All", "Descr", "Name", "Alias", "Type", "Speed", "Mac", "Admin",
// "Oper", "InOctets", "InPkts", "InMcast", "InBcast", "InErrors", "OutOctets", "OutPkts",
// "OutMcast", "OutBcast", "OutErrors"
func (sd *deviceUbiquiti) IfInfo(targets []string, idx ...string) (map[string]*ifInfo, error) {
	out := make(map[string]*ifInfo)

	idxs := make(map[string]bool)
	for _, i := range idx {
		idxs[i] = true
	}

	const descrOid = ".1.3.6.1.4.1.41112.1.5.7.2.1.2"
	const operOid = ".1.3.6.1.4.1.41112.1.5.7.2.1.3"

	rSrc := map[string]bool{
		"interfaces": false,
		"statistics": false,
		"oper":       false,
		"descr":      false,
	}

	for _, t := range targets {
		switch t {
		case "All":
			rSrc["interfaces"] = true
			rSrc["statistics"] = true
			rSrc["oper"] = true
			rSrc["descr"] = true
			continue
		case "Descr":
			rSrc["descr"] = true
		case "Name":
			rSrc["descr"] = true
		case "Alias":
			rSrc["descr"] = true
			rSrc["interfaces"] = true
		case "Type":
			rSrc["descr"] = true
		case "Speed":
			rSrc["descr"] = true
			rSrc["interfaces"] = true
		case "Mac":
			rSrc["descr"] = true
			rSrc["interfaces"] = true
		case "Admin":
			rSrc["descr"] = true
			rSrc["interfaces"] = true
		case "Oper":
			rSrc["descr"] = true
			rSrc["oper"] = true
		case "InOctets":
			rSrc["descr"] = true
			rSrc["statistics"] = true
		case "InPkts":
			rSrc["descr"] = true
			rSrc["statistics"] = true
		case "InMcast":
			rSrc["descr"] = true
			rSrc["statistics"] = true
		case "InBcast":
			rSrc["descr"] = true
			rSrc["statistics"] = true
		case "InDiscards":
			rSrc["descr"] = true
			rSrc["statistics"] = true
		case "InErrors":
			rSrc["descr"] = true
			rSrc["statistics"] = true
		case "OutOctets":
			rSrc["descr"] = true
			rSrc["statistics"] = true
		case "OutPkts":
			rSrc["descr"] = true
			rSrc["statistics"] = true
		case "OutMast":
			rSrc["descr"] = true
			rSrc["statistics"] = true
		case "OutBcast":
			rSrc["descr"] = true
			rSrc["statistics"] = true
		case "OutDiscards":
			rSrc["descr"] = true
			rSrc["statistics"] = true
		case "OutErrors":
			rSrc["descr"] = true
			rSrc["statistics"] = true
		}
	}

	var rawDescr snmphelper.SnmpOut
	var rawOper snmphelper.SnmpOut
	var rawIfInfo *UbiOltInterfaces
	var rawIfStats *UbiOltStatistics

	if rSrc["descr"] {
		r, err := sd.snmpSession.Walk(descrOid, true, true)
		if err != nil && sd.handleErr(descrOid, err) {
			return out, err
		}

		if idx != nil {
			for i := range r {
				if _, present := idxs[i]; !present {
					delete(r, i)
				}
			}
		}

		rawDescr = r
	}

	if rSrc["oper"] {
		r, err := sd.snmpSession.Walk(operOid, true, true)
		if err != nil && sd.handleErr(operOid, err) {
			return out, err
		}

		rawOper = r
	}

	if rSrc["interfaces"] {
		r, err := sd.oltIfInfo()
		if err != nil {
			return out, err
		}

		rawIfInfo = r
	}

	if rSrc["statistics"] {
		r, err := sd.oltStatistics()
		if err != nil {
			return out, err
		}

		rawIfStats = r
	}

	for i, d := range rawDescr {
		wDescr := strings.Replace(d.OctetString, "+", "", -1)
		out[i] = new(ifInfo)

		for _, t := range targets {
			if t == "All" || t == "Descr" {
				out[i].Descr.Value = wDescr
				out[i].Descr.IsSet = true
			}
			if t == "All" || t == "Name" {
				out[i].Name.Value = d.OctetString
				out[i].Name.IsSet = true
			}
			if t == "All" || t == "Oper" {
				out[i].Oper.Value = rawOper[i].Integer
				out[i].Oper.IsSet = true
				out[i].OperStr.Value = IfStatStr(rawOper[i].Integer)
				out[i].OperStr.IsSet = true
			}
			if t == "All" || t == "Alias" {
				for _, wi := range *rawIfInfo {
					if *wi.Identification.ID == wDescr {
						out[i].Alias.Value = *wi.Identification.Name
						out[i].Alias.IsSet = true
					}
				}
			}
			if t == "All" || t == "Type" {
				out[i].Type.Value = 6
				out[i].Type.IsSet = true
				out[i].TypeStr.Value = IfTypeStr(6)
				out[i].TypeStr.IsSet = true
			}
			if t == "All" || t == "Speed" {
				for _, wi := range *rawIfInfo {
					if *wi.Identification.ID == wDescr {
						speed := strings.Split(*wi.Status.CurrentSpeed, "-")
						si, _ := strconv.Atoi(speed[0])
						out[i].Speed.Value = uint64(si) * 1000000
						out[i].Speed.IsSet = true
					}
				}
			}
			if t == "All" || t == "Mac" {
				for _, wi := range *rawIfInfo {
					if *wi.Identification.ID == wDescr {
						out[i].Mac.Value = *wi.Identification.Mac
						out[i].Mac.IsSet = true
					}
				}
			}
			if t == "All" || t == "Admin" {
				for _, wi := range *rawIfInfo {
					if *wi.Identification.ID == wDescr {
						var stat int64 = 2
						if *wi.Status.Enabled {
							stat = 1
							out[i].Admin.Value = stat
							out[i].Admin.IsSet = true
							out[i].AdminStr.Value = IfStatStr(stat)
							out[i].AdminStr.IsSet = true
						}
					}
				}
			}
			if t == "All" || t == "InOctets" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if id.ID == wDescr {
							out[i].InOctets.Value = id.Statistics.RxBytes
							out[i].InOctets.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "InPkts" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if id.ID == wDescr {
							out[i].InPkts.Value = id.Statistics.RxPackets
							out[i].InPkts.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "InMcast" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if id.ID == wDescr {
							out[i].InMcast.Value = id.Statistics.RxMulticast
							out[i].InMcast.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "InBcast" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if id.ID == wDescr {
							out[i].InBcast.Value = id.Statistics.RxBroadcast
							out[i].InBcast.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "InErrors" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if id.ID == wDescr {
							out[i].InErrors.Value = id.Statistics.RxErrors
							out[i].InErrors.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "OutOctets" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if id.ID == wDescr {
							out[i].OutOctets.Value = id.Statistics.TxBytes
							out[i].OutOctets.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "OutPkts" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if id.ID == wDescr {
							out[i].OutPkts.Value = id.Statistics.TxPackets
							out[i].OutPkts.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "OutMcast" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if id.ID == wDescr {
							out[i].OutMcast.Value = id.Statistics.TxMulticast
							out[i].OutMcast.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "OutBcast" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if id.ID == wDescr {
							out[i].OutBcast.Value = id.Statistics.TxBroadcast
							out[i].OutBcast.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "OutErrors" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if id.ID == wDescr {
							out[i].OutErrors.Value = id.Statistics.TxErrors
							out[i].OutErrors.IsSet = true
						}
					}
					break
				}
			}
		}
	}

	return out, nil
}

// Set Interface Admin status
// set - map of ifIndexes and their states (up|down)
func (sd *deviceUbiquiti) SetIfAdmStat(set map[string]string) error {
	ifInfo, err := sd.IfInfo([]string{"Descr", "Admin"})
	if err != nil {
		return err
	}

	rawIfInfo, err := sd.oltIfInfo()
	if err != nil {
		return err
	}

	states := map[string]bool{
		"up":   true,
		"down": false,
	}

	dSet := make(map[string]bool)
	for idx, state := range set {
		s, ok := states[state]
		if !ok {
			return fmt.Errorf("interface state %s is not valid", state)
		}

		info, ok := ifInfo[idx]
		if !ok {
			return fmt.Errorf("interface with ifindex %s not found", idx)
		}

		dSet[info.Descr.Value] = s
	}

	var newPonIfs []UbiOltInterfacePonSet

	for _, rinfo := range *rawIfInfo {

		d := rinfo.Identification.ID
		if v, ok := dSet[*d]; ok {
			if v != *rinfo.Status.Enabled {
				if strings.HasPrefix(*d, "pon") {
					newPonIf := new(UbiOltInterfacePonSet)
					newPonIf.Addresses = rinfo.Addresses
					newPonIf.Identification = rinfo.Identification
					newPonIf.Pon = rinfo.Pon
					newPonIf.Status = rinfo.Status
					newPonIf.Status.Enabled = &v
					newPonIfs = append(newPonIfs, *newPonIf)
				}
			}
		}
	}

	if len(newPonIfs) > 0 {
		jsonData, err := json.Marshal(newPonIfs)
		if err != nil {
			return err
		}

		if err := sd.WebAuth(sd.webSession.cred); err != nil {
			return fmt.Errorf("error: WebAuth - %s", err)
		}

		_, err = sd.WebApiPut("interfaces", jsonData)
		if err != nil {
			return err
		}

		err = sd.WebLogout()
		if err != nil {
			return fmt.Errorf("errors: WebLogout - %s", err)
		}
	}

	return err
}

// Set Interface Alias
// set - map of ifIndexes and related ifAliases
func (sd *deviceUbiquiti) SetIfAlias(set map[string]string) (err error) {
	ifInfo, err := sd.IfInfo([]string{"Descr", "Alias"})
	if err != nil {
		return err
	}

	rawIfInfo, err := sd.oltIfInfo()
	if err != nil {
		return err
	}

	dSet := make(map[string]string)
	for idx, alias := range set {
		info, ok := ifInfo[idx]
		if !ok {
			return fmt.Errorf("interface with ifindex %s not found", idx)
		}

		dSet[info.Descr.Value] = alias
	}

	var newPonIfs []UbiOltInterfacePonSet
	var newPortIfs []UbiOltInterfacePortSet

	for _, rinfo := range *rawIfInfo {

		d := rinfo.Identification.ID
		if v, ok := dSet[*d]; ok {
			if v != *rinfo.Identification.Name {
				if strings.HasPrefix(*d, "pon") {
					newPonIf := new(UbiOltInterfacePonSet)
					newPonIf.Addresses = rinfo.Addresses
					newPonIf.Identification = rinfo.Identification
					newPonIf.Pon = rinfo.Pon
					newPonIf.Status = rinfo.Status
					newPonIf.Identification.Name = &v
					newPonIfs = append(newPonIfs, *newPonIf)
				} else if strings.HasPrefix(*d, "sfp") {
					newPortIf := new(UbiOltInterfacePortSet)
					newPortIf.Addresses = rinfo.Addresses
					newPortIf.Identification = rinfo.Identification
					newPortIf.Port = rinfo.Port
					newPortIf.Status = rinfo.Status
					newPortIf.Identification.Name = &v
					newPortIfs = append(newPortIfs, *newPortIf)
				}
			}
		}
	}

	if len(newPonIfs) > 0 || len(newPortIfs) > 0 {
		if err := sd.WebAuth(sd.webSession.cred); err != nil {
			return fmt.Errorf("error: WebAuth - %s", err)
		}

		defer func() {
			if err2 := sd.WebLogout(); err2 != nil {
				if err != nil {
					err = fmt.Errorf("%w; WebLogout - %s", err, err2)
				} else {
					err = fmt.Errorf("error: WebLogout - %s", err2)
				}
			}
		}()
	}

	if len(newPonIfs) > 0 {
		jsonData, err := json.Marshal(newPonIfs)
		if err != nil {
			return err
		}

		if _, err := sd.WebApiPut("interfaces", jsonData); err != nil {
			return fmt.Errorf("errors: WebApiPut - %s", err)
		}
	}

	if len(newPortIfs) > 0 {
		jsonData, err := json.Marshal(newPortIfs)
		if err != nil {
			return err
		}

		if _, err := sd.WebApiPut("interfaces", jsonData); err != nil {
			return fmt.Errorf("errors: WebApiPut - %s", err)
		}
	}

	return
}
