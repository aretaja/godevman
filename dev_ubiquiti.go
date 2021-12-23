package godevman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net"
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
type Address struct {
	Cidr    *string     `json:"cidr"`
	Origin  interface{} `json:"origin"`
	Type    *string     `json:"type"`
	Version *string     `json:"version"`
}

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
	Static     *bool         `json:"static"`
	Interfaces []interface{} `json:"interfaces"`
}

type UbiOltInterface struct {
	Addresses      []Address                     `json:"addresses"`
	Identification UbiOltInterfaceIdentification `json:"identification,omitempty"`
	Pon            UbiOltInterfaceSfp            `json:"pon,omitempty"`
	Status         UbiOltInterfaceStatus         `json:"status,omitempty"`
	Port           UbiOltInterfaceSfp            `json:"port,omitempty"`
	Lag            UbiOltInterfaceLag            `json:"lag,omitempty"`
}

type UbiOltInterfaces []UbiOltInterface

type UbiOltInterfacePonSet struct {
	Pon            UbiOltInterfaceSfp            `json:"pon,omitempty"`
	Identification UbiOltInterfaceIdentification `json:"identification,omitempty"`
	Status         UbiOltInterfaceStatus         `json:"status,omitempty"`
	Addresses      []Address                     `json:"addresses"`
}

type UbiOltInterfacePortSet struct {
	Port           UbiOltInterfaceSfp            `json:"port,omitempty"`
	Identification UbiOltInterfaceIdentification `json:"identification,omitempty"`
	Status         UbiOltInterfaceStatus         `json:"status,omitempty"`
	Addresses      []Address                     `json:"addresses"`
}

// Ubiquiti specific OLT statistics type used by device web API
type UbiOltStatistics []struct {
	Interfaces []struct {
		ID         *string `json:"id"`
		Name       *string `json:"name"`
		Statistics struct {
			RxBroadcast *uint64 `json:"rxBroadcast"`
			RxBytes     *uint64 `json:"rxBytes"`
			RxErrors    *uint64 `json:"rxErrors"`
			RxMulticast *uint64 `json:"rxMulticast"`
			RxPackets   *uint64 `json:"rxPackets"`
			RxRate      *uint64 `json:"rxRate"`
			TxBroadcast *uint64 `json:"txBroadcast"`
			TxBytes     *uint64 `json:"txBytes"`
			TxErrors    *uint64 `json:"txErrors"`
			TxMulticast *uint64 `json:"txMulticast"`
			TxPackets   *uint64 `json:"txPackets"`
			TxRate      *uint64 `json:"txRate"`
		} `json:"statistics,omitempty"`
	} `json:"interfaces"`
	Device struct {
		RAM struct {
			Free  *int `json:"free"`
			Total *int `json:"total"`
			Usage *int `json:"usage"`
		} `json:"ram"`
		FanSpeeds []struct {
			Value *float64 `json:"value"`
		} `json:"fanSpeeds"`
		Power []struct {
			PsuType   *string  `json:"psuType"`
			Current   *float64 `json:"current,omitempty"`
			Power     *float64 `json:"power,omitempty"`
			Voltage   *float64 `json:"voltage,omitempty"`
			Connected *bool    `json:"connected"`
		} `json:"power"`
		Signals []interface{} `json:"signals"`
		Storage []struct {
			Name        *string  `json:"name"`
			SysName     *string  `json:"sysName"`
			Type        *string  `json:"type"`
			Size        *int     `json:"size"`
			Temperature *float64 `json:"temperature"`
			Used        *int     `json:"used"`
		} `json:"storage"`
		Temperatures []struct {
			Value *float64 `json:"value"`
		} `json:"temperatures"`
		CPU []struct {
			Identifier  *string  `json:"identifier"`
			Temperature *float64 `json:"temperature"`
			Usage       *int     `json:"usage"`
		} `json:"cpu"`
		Uptime int `json:"uptime"`
	} `json:"device"`
	Timestamp int64 `json:"timestamp"`
}

// Ubiquiti specific VLANs type used by device web API
type UbiOltVlans struct {
	Trunks []interface{} `json:"trunks"`
	Vlans  []struct {
		Name          string `json:"name"`
		Type          string `json:"type"`
		Participation []struct {
			Interface struct {
				ID string `json:"id"`
			} `json:"interface"`
			Mode string `json:"mode"`
		} `json:"participation"`
		ID int `json:"id"`
	} `json:"vlans"`
}

// Ubiquiti specific ONU info type used by device web API
type UbiOnuInfo struct {
	Router struct{} `json:"router"`
	System struct {
		CPU         *float64 `json:"cpu"`
		Mem         *float64 `json:"mem"`
		Temperature struct {
			CPU *float64 `json:"cpu"`
		} `json:"temperature"`
		Uptime  *uint64  `json:"uptime"`
		Voltage *float64 `json:"voltage"`
	} `json:"system"`
	Statistics struct {
		RxBytes *int64 `json:"rxBytes"`
		RxRate  *int   `json:"rxRate"`
		TxBytes *int64 `json:"txBytes"`
		TxRate  *int   `json:"txRate"`
	} `json:"statistics"`
	Mac             *string  `json:"mac"`
	Error           *string  `json:"error"`
	FirmwareHash    *string  `json:"firmwareHash"`
	Serial          *string  `json:"serial"`
	Connected       *bool    `json:"connected"`
	FirmwareVersion *string  `json:"firmwareVersion"`
	TxPower         *float64 `json:"txPower"`
	OltPort         *int     `json:"oltPort"`
	LaserBias       *float64 `json:"laserBias"`
	RxPower         *float64 `json:"rxPower"`
	Distance        *int     `json:"distance"`
	ConnectionTime  *uint64  `json:"connectionTime"`
	Authorized      *bool    `json:"authorized"`
	UpgradeStatus   struct {
		FailureReason string `json:"failureReason"`
		Status        string `json:"status"`
	} `json:"upgradeStatus"`
	Ports []struct {
		ID      *string `json:"id"`
		Speed   *string `json:"speed"`
		Plugged *bool   `json:"plugged"`
	} `json:"ports"`
}

type UbiOnusInfo []UbiOnuInfo

// Ubiquiti specific ONU settings type used by device web API
type UbiOnuSettings struct {
	Services struct {
		HTTPPort             *int  `json:"httpPort"`
		SSHPort              *int  `json:"sshPort"`
		TelnetPort           *int  `json:"telnetPort"`
		SSHEnabled           *bool `json:"sshEnabled"`
		TelnetEnabled        *bool `json:"telnetEnabled"`
		UbntDiscoveryEnabled *bool `json:"ubntDiscoveryEnabled"`
	} `json:"services"`
	BandwidthLimit struct {
		Download struct {
			Enabled *bool `json:"enabled"`
			Limit   *int  `json:"limit"`
		} `json:"download"`
		Upload struct {
			Enabled *bool `json:"enabled"`
			Limit   *int  `json:"limit"`
		} `json:"upload"`
	} `json:"bandwidthLimit"`
	Enabled        *bool   `json:"enabled"`
	Name           *string `json:"name"`
	LanAddress     *string `json:"lanAddress"`
	Model          *string `json:"model"`
	Mode           *string `json:"mode"`
	AdminPassword  *string `json:"adminPassword"`
	LanProvisioned *bool   `json:"lanProvisioned"`
	Notes          *string `json:"notes"`
	Serial         *string `json:"serial"`
	Wifi           struct {
		Channel       *string  `json:"channel"`
		ChannelWidth  *string  `json:"channelWidth"`
		Country       *string  `json:"country"`
		CountryListID *string  `json:"countryListId"`
		TxPower       *float64 `json:"txPower"`
		Enabled       *bool    `json:"enabled"`
		Provisioned   *bool    `json:"provisioned"`
		Networks      []struct {
			AuthMode *string `json:"authMode"`
			Key      *string `json:"key"`
			Ssid     *string `json:"ssid"`
			HideSSID *bool   `json:"hideSSID"`
		} `json:"networks"`
	} `json:"wifi"`
	Ports []struct {
		ID    *string `json:"id"`
		Speed *string `json:"speed"`
	} `json:"ports"`
	BridgeMode struct {
		Ports []struct {
			Port         *string `json:"port"`
			NativeVLAN   *int    `json:"nativeVLAN"`
			IncludeVLANs []int   `json:"includeVLANs"`
		} `json:"ports"`
	} `json:"bridgeMode"`
	RouterMode struct {
		RouterAdvertisement struct {
			Mode   *string `json:"mode"`
			Prefix *string `json:"prefix"`
		} `json:"routerAdvertisement"`
		DhcpPool struct {
			RangeStart *string `json:"rangeStart"`
			RangeStop  *string `json:"rangeStop"`
		} `json:"dhcpPool"`
		PppoePassword    *string       `json:"pppoePassword"`
		DhcpServerMode   *string       `json:"dhcpServerMode"`
		WanMode6         *string       `json:"wanMode6"`
		PppoeUser        *string       `json:"pppoeUser"`
		WanMode          *string       `json:"wanMode"`
		Gateway          *string       `json:"gateway"`
		Gateway6         *string       `json:"gateway6"`
		DhcpRelay        *string       `json:"dhcpRelay"`
		LanAddress6      *string       `json:"lanAddress6"`
		LanMode6         *string       `json:"lanMode6"`
		WanAddress6      *string       `json:"wanAddress6"`
		PppoeMode        *string       `json:"pppoeMode"`
		WanAddress       *string       `json:"wanAddress"`
		FirewallEnabled6 *bool         `json:"firewallEnabled6"`
		Ipv6Enabled      *bool         `json:"ipv6Enabled"`
		DhcpLeaseTime    *int          `json:"dhcpLeaseTime"`
		WanVLAN          *int          `json:"wanVLAN"`
		UpnpEnabled      *bool         `json:"upnpEnabled"`
		WanAccessBlocked *bool         `json:"wanAccessBlocked"`
		DNSProxyEnabled  *bool         `json:"dnsProxyEnabled"`
		DNSResolvers     []interface{} `json:"dnsResolvers"`
		PortForwards     []interface{} `json:"portForwards"`
		Nat              struct {
			Ftp  bool `json:"ftp"`
			Pptp bool `json:"pptp"`
			Rtsp bool `json:"rtsp"`
			Sip  bool `json:"sip"`
		} `json:"nat"`
	} `json:"routerMode"`
}

type UbiOnusSettings []UbiOnuSettings

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

// Get OLT VLAN info via web API.
func (sd *deviceUbiquiti) oltVlans() (*UbiOltVlans, error) {
	// return from cache if allowed and cache is present
	if sd.useCache {
		if x, found := sd.cache.Get("oltVlans"); found {
			return x.(*UbiOltVlans), nil
		}
	}

	if err := sd.WebAuth(sd.webSession.cred); err != nil {
		return nil, fmt.Errorf("error: WebAuth - %s", err)
	}

	body, err := sd.WebApiGet("vlans")
	if err != nil {
		return nil, fmt.Errorf("get request from device api failed: %s", err)
	}

	err = sd.WebLogout()
	if err != nil {
		return nil, fmt.Errorf("errors: WebLogout - %s", err)
	}

	info := new(UbiOltVlans)
	err = json.Unmarshal(body, info)
	if err != nil {
		return nil, fmt.Errorf("unmarshal OLT VLANS failed: %s", err)
	}

	if info == nil {
		return nil, fmt.Errorf("no OLT VLANS")
	}

	// save to cache
	sd.cache.Set("oltVlans", info, cache.DefaultExpiration)

	return info, nil
}

// Get all ONU info via web API.
func (sd *deviceUbiquiti) oltOnus() (*UbiOnusInfo, error) {
	// return from cache if allowed and cache is present
	if sd.useCache {
		if x, found := sd.cache.Get("oltOnus"); found {
			return x.(*UbiOnusInfo), nil
		}
	}

	if err := sd.WebAuth(sd.webSession.cred); err != nil {
		return nil, fmt.Errorf("error: WebAuth - %s", err)
	}

	body, err := sd.WebApiGet("gpon/onus")
	if err != nil {
		return nil, fmt.Errorf("get request from device api failed: %s", err)
	}

	err = sd.WebLogout()
	if err != nil {
		return nil, fmt.Errorf("errors: WebLogout - %s", err)
	}

	info := new(UbiOnusInfo)
	err = json.Unmarshal(body, info)
	if err != nil {
		return nil, fmt.Errorf("unmarshal ONU info failed: %s", err)
	}

	if info == nil {
		return nil, fmt.Errorf("no ONU info")
	}

	// save to cache
	sd.cache.Set("oltOnus", info, cache.DefaultExpiration)

	return info, nil
}

// Get all ONU settings via web API.
func (sd *deviceUbiquiti) oltOnuSettings() (*UbiOnusSettings, error) {
	// return from cache if allowed and cache is present
	if sd.useCache {
		if x, found := sd.cache.Get("oltOnuSettings"); found {
			return x.(*UbiOnusSettings), nil
		}
	}

	if err := sd.WebAuth(sd.webSession.cred); err != nil {
		return nil, fmt.Errorf("error: WebAuth - %s", err)
	}

	body, err := sd.WebApiGet("gpon/onus/settings")
	if err != nil {
		return nil, fmt.Errorf("get request from device api failed: %s", err)
	}

	err = sd.WebLogout()
	if err != nil {
		return nil, fmt.Errorf("errors: WebLogout - %s", err)
	}

	info := new(UbiOnusSettings)
	err = json.Unmarshal(body, info)
	if err != nil {
		return nil, fmt.Errorf("unmarshal ONU settings failed: %s", err)
	}

	if info == nil {
		return nil, fmt.Errorf("no ONU settings")
	}

	// save to cache
	sd.cache.Set("oltOnuSettings", info, cache.DefaultExpiration)

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
						if *id.ID == wDescr && id.Statistics.RxBytes != nil {
							out[i].InOctets.Value = *id.Statistics.RxBytes
							out[i].InOctets.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "InPkts" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if *id.ID == wDescr && id.Statistics.RxPackets != nil {
							out[i].InPkts.Value = *id.Statistics.RxPackets
							out[i].InPkts.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "InMcast" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if *id.ID == wDescr && id.Statistics.RxMulticast != nil {
							out[i].InMcast.Value = *id.Statistics.RxMulticast
							out[i].InMcast.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "InBcast" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if *id.ID == wDescr && id.Statistics.RxBroadcast != nil {
							out[i].InBcast.Value = *id.Statistics.RxBroadcast
							out[i].InBcast.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "InErrors" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if *id.ID == wDescr && id.Statistics.RxErrors != nil {
							out[i].InErrors.Value = *id.Statistics.RxErrors
							out[i].InErrors.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "OutOctets" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if *id.ID == wDescr && id.Statistics.TxBytes != nil {
							out[i].OutOctets.Value = *id.Statistics.TxBytes
							out[i].OutOctets.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "OutPkts" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if *id.ID == wDescr && id.Statistics.TxPackets != nil {
							out[i].OutPkts.Value = *id.Statistics.TxPackets
							out[i].OutPkts.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "OutMcast" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if *id.ID == wDescr && id.Statistics.TxMulticast != nil {
							out[i].OutMcast.Value = *id.Statistics.TxMulticast
							out[i].OutMcast.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "OutBcast" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if *id.ID == wDescr && id.Statistics.TxBroadcast != nil {
							out[i].OutBcast.Value = *id.Statistics.TxBroadcast
							out[i].OutBcast.IsSet = true
						}
					}
					break
				}
			}
			if t == "All" || t == "OutErrors" {
				for _, wi := range *rawIfStats {
					for _, id := range wi.Interfaces {
						if *id.ID == wDescr && id.Statistics.TxErrors != nil {
							out[i].OutErrors.Value = *id.Statistics.TxErrors
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

// Get info from device web API
// Returns vlan id-s and names
func (sd *deviceUbiquiti) D1qVlans() (map[string]string, error) {
	var out = make(map[string]string)

	rawVlans, err := sd.oltVlans()
	if err != nil {
		return out, err
	}

	for _, v := range rawVlans.Vlans {
		out[strconv.Itoa(v.ID)] = v.Name
	}

	return out, err
}

// Get info from device web API
// Returns vlan port relations
func (sd *deviceUbiquiti) D1qVlanInfo() (map[string]*d1qVlanInfo, error) {
	out := make(map[string]*d1qVlanInfo)

	iInfo, err := sd.IfInfo([]string{"Descr"})
	if err != nil {
		return out, err
	}

	vInfo, err := sd.oltVlans()
	if err != nil {
		return out, err
	}

	descr := make(map[string]string)
	for i, data := range iInfo {
		if data.Descr.IsSet {
			descr[data.Descr.Value] = i
		}
	}

	for _, v := range vInfo.Vlans {
		vidStr := strconv.Itoa(v.ID)
		out[vidStr] = new(d1qVlanInfo)
		out[vidStr].Name = v.Name
		out[vidStr].Ports = make(map[int]*d1qVlanBrPort)
		for _, p := range v.Participation {
			if i, ok := descr[p.Interface.ID]; ok {
				pId, _ := strconv.Atoi(i)
				out[vidStr].Ports[pId] = new(d1qVlanBrPort)
				out[vidStr].Ports[pId].IfIdx = pId
				if p.Mode != "tagged" {
					out[vidStr].Ports[pId].UnTag = true
				}
			}
		}
	}

	return out, err
}

// Get info via web API
func (sd *deviceUbiquiti) IpInfo(ip ...string) (map[string]*ipInfo, error) {
	out := make(map[string]*ipInfo)

	ifInfo, err := sd.IfInfo([]string{"Descr"})
	if err != nil {
		return out, err
	}

	rawIfInfo, err := sd.oltIfInfo()
	if err != nil {
		return out, err
	}

	descr := make(map[string]string)
	for i, data := range ifInfo {
		if data.Descr.IsSet {
			descr[data.Descr.Value] = i
		}
	}

	for _, i := range *rawIfInfo {
		if len(i.Addresses) > 0 {
			for _, a := range i.Addresses {
				if idx, ok := descr[*i.Identification.ID]; ok && *a.Version == "v4" {
					idxStr, _ := strconv.Atoi(idx)

					ipAddr, net, err := net.ParseCIDR(*a.Cidr)
					if err != nil {
						continue
					}

					ipStr := ipAddr.String()

					m := net.Mask
					if len(m) != 4 {
						continue
					}

					mask := fmt.Sprintf("%d.%d.%d.%d", m[0], m[1], m[2], m[3])

					out[ipStr] = new(ipInfo)
					out[ipStr].IfIdx = int64(idxStr)
					out[ipStr].Mask = mask
				}
			}
		}
	}

	if len(ip) > 0 && len(out) > 0 {
		filter := make(map[string]bool)
		for _, i := range ip {
			filter[i] = true
		}

		for i := range out {
			if _, ok := filter[i]; !ok {
				delete(out, i)
			}
		}

		if len(out) == 0 {
			return nil, fmt.Errorf("none of requested ips found")
		}
	}

	return out, err
}

// Get IP Interface info
func (sd *deviceUbiquiti) IpIfInfo(ip ...string) (map[string]*ipIfInfo, error) {
	out := make(map[string]*ipIfInfo)

	ipInfo, err := sd.IpInfo(ip...)
	if err != nil {
		return out, err
	}

	ifInfo, err := sd.IfInfo([]string{"Descr", "Alias"})
	if err != nil {
		return out, err
	}

	for i, d := range ipInfo {
		out[i] = new(ipIfInfo)
		ifIdxStr := strconv.FormatInt(int64(d.IfIdx), 10)

		var descr, alias string
		ifdata, ok := ifInfo[ifIdxStr]
		if !ok {
			descr = "unkn_" + ifIdxStr
		} else {
			descr = ifdata.Descr.Value
			alias = ifdata.Alias.Value
		}

		out[i].ipInfo = *d
		out[i].Descr = descr
		out[i].Alias = alias
	}

	return out, err
}

// Valid targets values: "All", "Fan", "Power", "Temp", "Ram", "Cpu", "Storage"
func (sd *deviceUbiquiti) Sensors(targets []string) (map[string]map[string]map[string]sensorVal, error) {
	out := make(map[string]map[string]map[string]sensorVal)

	rawStats, err := sd.oltStatistics()
	if err != nil {
		return out, err
	}

	t := map[string]bool{
		"Fan":     false,
		"Power":   false,
		"Temp":    false,
		"Ram":     false,
		"Cpu":     false,
		"Storage": false,
	}

	for _, v := range targets {
		if v == "All" {
			for k := range t {
				t[k] = true
			}
		} else if _, ok := t[v]; ok {
			t[v] = true
		}
	}

	for _, s := range *rawStats {
		if (t["All"] || t["Fan"]) && len(s.Device.FanSpeeds) > 0 {
			if out["Fan"] == nil {
				out["Fan"] = make(map[string]map[string]sensorVal)
			}

			for i, v := range s.Device.FanSpeeds {
				sName := "Sensor" + strconv.Itoa(i+1)
				sValue := sensorVal{
					Unit:    "rpm",
					Divisor: 100,
					Value:   uint64(*v.Value * 100),
					IsSet:   true,
				}

				if out["Fan"]["Speed"] == nil {
					out["Fan"]["Speed"] = make(map[string]sensorVal)
				}

				out["Fan"]["Speed"][sName] = sValue
			}
		}

		if (t["All"] || t["Temp"]) && len(s.Device.Temperatures) > 0 {
			if out["Temp"] == nil {
				out["Temp"] = make(map[string]map[string]sensorVal)
			}

			for i, v := range s.Device.Temperatures {
				sName := "Sensor" + strconv.Itoa(i+1)
				sValue := sensorVal{
					Unit:    "°C",
					Divisor: 100,
					Value:   uint64(math.Abs(*v.Value * 100)),
					IsSet:   true,
				}
				if *v.Value < 0 {
					sValue.Divisor = -100
				}

				if out["Temp"]["Chasis"] == nil {
					out["Temp"]["Chasis"] = make(map[string]sensorVal)
				}

				out["Temp"]["Chasis"][sName] = sValue
			}
		}

		if (t["All"] || t["Power"]) && len(s.Device.Power) > 0 {
			if out["Power"] == nil {
				out["Power"] = make(map[string]map[string]sensorVal)
			}

			for i, v := range s.Device.Power {
				sName := *v.PsuType + strconv.Itoa(i+1)

				if out["Power"][sName] == nil {
					out["Power"][sName] = make(map[string]sensorVal)
				}

				if v.Current != nil {
					sValue := sensorVal{
						Unit:    "A",
						Divisor: 100,
						Value:   uint64(*v.Current * 100),
						IsSet:   true,
					}
					out["Power"][sName]["Current"] = sValue
				}

				if v.Voltage != nil {
					sValue := sensorVal{
						Unit:    "V",
						Divisor: 100,
						Value:   uint64(*v.Voltage * 100),
						IsSet:   true,
					}
					out["Power"][sName]["Voltage"] = sValue
				}

				if v.Power != nil {
					sValue := sensorVal{
						Unit:    "W",
						Divisor: 100,
						Value:   uint64(*v.Power * 100),
						IsSet:   true,
					}
					out["Power"][sName]["Power"] = sValue
				}

				if v.Connected != nil {
					sValue := sensorVal{
						Bool:  *v.Connected,
						IsSet: true,
					}
					out["Power"][sName]["Connected"] = sValue
				}
			}
		}

		if (t["All"] || t["Cpu"]) && len(s.Device.CPU) > 0 {
			if out["Cpu"] == nil {
				out["Cpu"] = make(map[string]map[string]sensorVal)
			}

			for _, v := range s.Device.CPU {
				if out["Cpu"][*v.Identifier] == nil {
					out["Cpu"][*v.Identifier] = make(map[string]sensorVal)
				}

				if v.Temperature != nil {
					sValue := sensorVal{
						Unit:    "°C",
						Divisor: 100,
						Value:   uint64(math.Abs(*v.Temperature * 100)),
						IsSet:   true,
					}
					if *v.Temperature < 0 {
						sValue.Divisor = -100
					}

					out["Cpu"][*v.Identifier]["Temp"] = sValue
				}

				if v.Usage != nil {
					sValue := sensorVal{
						Unit:    "%",
						Divisor: 100,
						Value:   uint64(*v.Usage * 100),
						IsSet:   true,
					}
					out["Cpu"][*v.Identifier]["Usage"] = sValue
				}
			}
		}

		if (t["All"] || t["Storage"]) && len(s.Device.Storage) > 0 {
			if out["Storage"] == nil {
				out["Storage"] = make(map[string]map[string]sensorVal)
			}

			for _, v := range s.Device.Storage {
				if out["Storage"][*v.Name] == nil {
					out["Storage"][*v.Name] = make(map[string]sensorVal)
				}

				if v.Size != nil {
					sValue := sensorVal{
						Unit:    "B",
						Divisor: 1,
						Value:   uint64(*v.Size),
						IsSet:   true,
					}
					out["Storage"][*v.Name]["Size"] = sValue
				}

				if v.SysName != nil {
					sValue := sensorVal{
						String: *v.SysName,
						IsSet:  true,
					}
					out["Storage"][*v.Name]["SysName"] = sValue
				}

				if v.Temperature != nil {
					sValue := sensorVal{
						Unit:    "°C",
						Divisor: 100,
						Value:   uint64(math.Abs(*v.Temperature * 100)),
						IsSet:   true,
					}
					if *v.Temperature < 0 {
						sValue.Divisor = -100
					}

					out["Storage"][*v.Name]["Temp"] = sValue
				}

				if v.Used != nil {
					sValue := sensorVal{
						Unit:    "B",
						Divisor: 1,
						Value:   uint64(*v.Used),
						IsSet:   true,
					}
					out["Storage"][*v.Name]["Used"] = sValue
				}
			}
		}

		if t["All"] || t["Ram"] {
			if out["Ram"] == nil {
				out["Ram"] = make(map[string]map[string]sensorVal)
			}
			if out["Ram"]["Status"] == nil {
				out["Ram"]["Status"] = make(map[string]sensorVal)
			}

			if s.Device.RAM.Free != nil {
				sValue := sensorVal{
					Unit:    "B",
					Divisor: 1,
					Value:   uint64(*s.Device.RAM.Free),
					IsSet:   true,
				}
				out["Ram"]["Status"]["Free"] = sValue
			}

			if s.Device.RAM.Total != nil {
				sValue := sensorVal{
					Unit:    "B",
					Divisor: 1,
					Value:   uint64(*s.Device.RAM.Total),
					IsSet:   true,
				}
				out["Ram"]["Status"]["Total"] = sValue
			}

			if s.Device.RAM.Usage != nil {
				sValue := sensorVal{
					Unit:    "%",
					Divisor: 1,
					Value:   uint64(*s.Device.RAM.Usage),
					IsSet:   true,
				}
				out["Ram"]["Status"]["Usage"] = sValue
			}
		}
	}
	return out, err
}

// Get info from device web API
// Returns OLT's ONU info
func (sd *deviceUbiquiti) OnuInfo() (map[string]*onuInfo, error) {
	out := make(map[string]*onuInfo)

	oInfo, err := sd.oltOnus()
	if err != nil {
		return out, err
	}

	oSettings, err := sd.oltOnuSettings()
	if err != nil {
		return out, err
	}

	onus := make(map[string]UbiOnuInfo)
	for _, i := range *oInfo {
		if i.OltPort == nil {
			continue
		}
		onus[*i.Serial] = i
	}

	for _, s := range *oSettings {
		if i, ok := onus[*s.Serial]; ok {
			o := &onuInfo{
				OltPort: valString{
					Value: "pon" + strconv.Itoa(*i.OltPort),
					IsSet: true,
				},
			}

			if s.Model != nil {
				o.Model.Value = *s.Model
				o.Model.IsSet = true
			}
			if s.Name != nil {
				o.Name.Value = *s.Name
				o.Name.IsSet = true
			}
			if s.Enabled != nil {
				o.Enabled.Value = *s.Enabled
				o.Enabled.IsSet = true
			}
			if i.Mac != nil {
				o.Mac.Value = *i.Mac
				o.Mac.IsSet = true
			}
			if i.Error != nil {
				o.Error.Value = *i.Error
				o.Error.IsSet = true
			}
			if i.FirmwareVersion != nil {
				o.Version.Value = *i.FirmwareVersion
				o.Version.IsSet = true
			}
			if i.Connected != nil {
				o.Online.Value = *i.Connected
				o.Online.IsSet = true
			}
			if i.Statistics.TxBytes != nil {
				v := sensorVal{
					Unit:    "B",
					Divisor: 1,
					Value:   uint64(*i.Statistics.TxBytes),
					IsSet:   true,
				}

				o.TxBytes = v
			}
			if i.Statistics.RxBytes != nil {
				v := sensorVal{
					Unit:    "B",
					Divisor: 1,
					Value:   uint64(*i.Statistics.RxBytes),
					IsSet:   true,
				}

				o.RxBytes = v
			}
			if i.TxPower != nil {
				v := sensorVal{
					Unit:    "dBm",
					Divisor: 100,
					Value:   uint64(math.Abs(*i.TxPower * 100)),
					IsSet:   true,
				}
				if *i.TxPower < 0 {
					v.Divisor = -100
				}

				o.TxPower = v
			}
			if i.RxPower != nil {
				v := sensorVal{
					Unit:    "dBm",
					Divisor: 100,
					Value:   uint64(math.Abs(*i.RxPower * 100)),
					IsSet:   true,
				}
				if *i.RxPower < 0 {
					v.Divisor = -100
				}

				o.RxPower = v
			}
			if i.System.Mem != nil {
				v := sensorVal{
					Unit:    "%",
					Divisor: 1,
					Value:   uint64(*i.System.Mem),
					IsSet:   true,
				}

				o.Ram = v
			}
			if i.Distance != nil {
				v := sensorVal{
					Unit:    "m",
					Divisor: 1,
					Value:   uint64(*i.Distance),
					IsSet:   true,
				}

				o.Distance = v
			}
			if i.System.Temperature.CPU != nil {
				v := sensorVal{
					Unit:    "°C",
					Divisor: 100,
					Value:   uint64(math.Abs(*i.System.Temperature.CPU * 100)),
					IsSet:   true,
				}
				if *i.System.Temperature.CPU < 0 {
					v.Divisor = -100
				}

				o.CpuTemp = v
			}
			if i.System.CPU != nil {
				v := sensorVal{
					Unit:    "%",
					Divisor: 1,
					Value:   uint64(*i.System.CPU),
					IsSet:   true,
				}

				o.CpuUsage = v
			}
			if s.BandwidthLimit.Download.Limit != nil && s.BandwidthLimit.Download.Enabled != nil && *s.BandwidthLimit.Download.Enabled {
				v := sensorVal{
					Unit:    "B",
					Divisor: 1,
					Value:   uint64(*s.BandwidthLimit.Download.Limit),
					IsSet:   true,
				}

				o.DownLimit = v
			}
			if s.BandwidthLimit.Upload.Limit != nil && s.BandwidthLimit.Upload.Enabled != nil && *s.BandwidthLimit.Upload.Enabled {
				v := sensorVal{
					Unit:    "B",
					Divisor: 1,
					Value:   uint64(*s.BandwidthLimit.Upload.Limit),
					IsSet:   true,
				}

				o.Uplimit = v
			}
			if i.System.Uptime != nil {
				o.UpTime.Value = *i.System.Uptime
				o.UpTime.IsSet = true
				o.UpTimeStr.Value = UpTimeString(*i.System.Uptime*100, 0)
				o.UpTimeStr.IsSet = true
			}
			if i.ConnectionTime != nil {
				o.ConTime.Value = *i.ConnectionTime
				o.ConTime.IsSet = true
				o.ConTimeStr.Value = UpTimeString(*i.ConnectionTime*100, 0)
				o.ConTimeStr.IsSet = true
			}
			if i.Ports != nil {
				o.Ports = make(map[string]onuPort)
				for _, p := range i.Ports {
					pi := new(onuPort)
					if p.Plugged != nil {
						pi.Plugged.Value = *p.Plugged
						pi.Plugged.IsSet = true
					}
					if p.Speed != nil {
						pi.Speed.Value = *p.Speed
						pi.Speed.IsSet = true
					}
					if p.ID != nil {
						pi.Id.Value = *p.ID
						pi.Id.IsSet = true

						o.Ports[*p.ID] = *pi
					}
				}

				for _, p := range s.Ports {
					if p.ID != nil {
						if _, ok := o.Ports[*p.ID]; ok {
							pi := o.Ports[*p.ID]
							pi.Mode.Value = *p.Speed
							pi.Mode.IsSet = true
							o.Ports[*p.ID] = pi
						}
					}
				}

				for _, p := range s.BridgeMode.Ports {
					if p.Port != nil {
						if _, ok := o.Ports[*p.Port]; ok {
							pi := o.Ports[*p.Port]
							pi.NativeVlan.Value = *p.NativeVLAN
							pi.NativeVlan.IsSet = true
							pi.Vlans = append(pi.Vlans, *p.NativeVLAN)

							if p.IncludeVLANs != nil {
								pi.Vlans = append(pi.Vlans, p.IncludeVLANs...)
							}
							o.Ports[*p.Port] = pi
						}
					}
				}
			}

			out[*s.Serial] = o
		}
	}

	return out, err
}
