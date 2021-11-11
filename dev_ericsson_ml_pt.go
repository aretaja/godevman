package godevman

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PraserX/ipconv"
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
	client := sd.webSession.client
	if sd.webSession.client == nil {
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

// Login via web API and stores web session in deviceEricssonMlPt.webSession.client.
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

	sd.webSession.client = client

	return nil
}

// Logout via web API and delete web session from deviceEricssonMlPt.webSession.client.
// Use this after use of methods which are accessing restricted device web API.
func (sd *deviceEricssonMlPt) WebLogout() error {
	if sd.webSession.client == nil {
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

	sd.webSession.client = nil

	return nil
}

// TODO
func (sd *deviceEricssonMlPt) OspfAreaRouters() (map[string][]string, error) {
	return nil, fmt.Errorf("func OspfAreaRouters not implemented yet on this device type")
}

// TODO
func (sd *deviceEricssonMlPt) OspfAreaStatus() (map[string]string, error) {
	return nil, fmt.Errorf("func OspfAreaStatus not implemented yet on this device type")
}

// Get OSPF neighbour status
func (sd *deviceEricssonMlPt) OspfNbrStatus() (map[string]string, error) {
	if err := sd.WebAuth(sd.webSession.cred); err != nil {
		return nil, fmt.Errorf("error: WebAuth - %s", err)
	}

	body, err := sd.WebApiGet("CATEGORY=JSONREQUEST&OSPF_NEIGHBOUR")
	if err != nil {
		return nil, fmt.Errorf("get request from device api failed: %s", err)
	}

	err = sd.WebLogout()
	if err != nil {
		return nil, fmt.Errorf("errors: WebLogout - %s", err)
	}

	// OSPF info provided by MINI-LINK PT web API
	type ospfInfo struct {
		OspfNeighbours []struct {
			OspfNeighbour struct {
				BInterfaceName              string `json:"bInterfaceName"`
				IRouterID                   int    `json:"iRouterId"`
				WInterfaceIndex             int    `json:"wInterfaceIndex"`
				ENeighbourState             int    `json:"eNeighbourState"`
				EPermanence                 int    `json:"ePermanence"`
				BPriority                   int    `json:"bPriority"`
				BOptions                    int    `json:"bOptions"`
				LEvents                     int    `json:"lEvents"`
				LRetransmisssionQueueLength int    `json:"lRetransmisssionQueueLength"`
			} `json:"OSPF_NEIGHBOUR"`
			OspfNeighbourMoid struct {
				WClass          int `json:"wClass"`
				IRouterID       int `json:"iRouterId"`
				WInterfaceIndex int `json:"wInterfaceIndex"`
				INbrIPAddr      int `json:"iNbrIpAddr"`
			} `json:"OSPF_NEIGHBOUR_MOID"`
		} `json:"OSPF_NEIGHBOUR"`
	}

	info := &ospfInfo{}
	err = json.Unmarshal(body, info)
	if err != nil {
		return nil, fmt.Errorf("unmarshal ospf info failed: %s", err)
	}

	if info == nil || len(info.OspfNeighbours) == 0 {
		return nil, fmt.Errorf("no ospf info")
	}

	states := map[int]string{
		11: "down",
		12: "attempt",
		13: "init",
		14: "twoWay",
		15: "exchangeStart",
		16: "exchange",
		17: "loading",
		18: "full",
	}

	out := make(map[string]string)
	for _, i := range info.OspfNeighbours {
		ip := ipconv.IntToIPv4(uint32(i.OspfNeighbourMoid.INbrIPAddr))

		// Reverse ip slice
		for i, j := 0, len(ip)-1; i < j; i, j = i+1, j-1 {
			ip[i], ip[j] = ip[j], ip[i]
		}

		out[ip.String()] = states[i.OspfNeighbour.ENeighbourState]
	}

	return out, err
}

// Get RL neighbour info
func (sd *deviceEricssonMlPt) RlNbrInfo() (map[int]*map[string]string, error) {
	body, err := sd.WebApiGet("CATEGORY=JSONREQUEST&FE_STATUS_VIEW")
	if err != nil {
		return nil, fmt.Errorf("get request from device api failed: %s", err)
	}

	type feRlInfo struct {
		FeStatusView struct {
			CtPassive string `json:"CT_PASSIVE"`
			Software  struct {
				BRunningNR string `json:"bRunningNR"`
			} `json:"SOFTWARE"`
			System struct {
				BName string `json:"bName"`
			} `json:"SYSTEM"`
			SystemTimeInfo struct {
				TimeZoneOffset        string `json:"timeZoneOffset"`
				UpTime                int64  `json:"upTime"`
				CurrentTimestamp      int    `json:"currentTimestamp"`
				CurrentLocalTimestamp int    `json:"currentLocalTimestamp"`
				TimeZoneOffsetMinutes int    `json:"timeZoneOffsetMinutes"`
				IsDSTEnabled          bool   `json:"isDSTEnabled"`
			} `json:"SYSTEM_TIME_INFO"`
			InterfaceModuleInfo []struct {
				InterfaceModuleInfo struct {
					BVendorName string `json:"bVendorName"`
				} `json:"INTERFACE_MODULE_INFO"`
				InterfaceModuleInfoMoid struct {
					BSlot int `json:"bSlot"`
					BPort int `json:"bPort"`
				} `json:"INTERFACE_MODULE_INFO_MOID"`
			} `json:"INTERFACE_MODULE_INFO"`
			CtMember []struct {
				CtMember struct {
					BDistinguishedName string `json:"bDistinguishedName"`
					BSlotNumber        int    `json:"bSlotNumber"`
					BCtNumber          int    `json:"bCtNumber"`
				} `json:"CT_MEMBER"`
			} `json:"CT_MEMBER"`
			LanxPort []struct {
				LanxPortMoid struct {
					WClass int `json:"wClass"`
					BSlot  int `json:"bSlot"`
					BPort  int `json:"bPort"`
				} `json:"LANX_PORT_MOID"`
				LanxPort struct {
					EAdminStatus int `json:"eAdminStatus"`
				} `json:"LANX_PORT"`
			} `json:"LANX_PORT"`
			Xpic        []interface{} `json:"XPIC"`
			IPInterface []struct {
				Address string `json:"address"`
				Mask    string `json:"mask"`
				Ipv6    string `json:"ipv6"`
				Index   int    `json:"index"`
			} `json:"IP_INTERFACE"`
			CurrentAlarmsEntry []interface{} `json:"CURRENT_ALARMS_ENTRY"`
			CarrierTermination []struct {
				BActualOutputPower       string `json:"bActualOutputPower"`
				BDescription             string `json:"bDescription"`
				BActualXpi               string `json:"bActualXpi"`
				BActualInputPower        string `json:"bActualInputPower"`
				BDistinguishedName       string `json:"bDistinguishedName"`
				LTxFrequency             int    `json:"lTxFrequency"`
				ETxOperStatus            int    `json:"eTxOperStatus"`
				ETxAdminStatus           int    `json:"eTxAdminStatus"`
				WFrameID                 int    `json:"wFrameId"`
				SbSelectedMinOutputPower int    `json:"sbSelectedMinOutputPower"`
				SbSelectedMaxOutputPower int    `json:"sbSelectedMaxOutputPower"`
				EStatus                  int    `json:"eStatus"`
				EXpicStatus              int    `json:"eXpicStatus"`
				ECarrierID               int    `json:"eCarrierId"`
				LActualTxCapacity        int    `json:"lActualTxCapacity"`
				EActualTxAcm             int    `json:"eActualTxAcm"`
			} `json:"CARRIER_TERMINATION"`
			RadioLinkTerminal struct {
				BDistinguishedName string `json:"bDistinguishedName"`
				I6NeIpv6Address    string `json:"i6NeIpv6Address"`
				BNeName            string `json:"bNeName"`
				BID                string `json:"bId"`
				INeIPAddress       int    `json:"iNeIpAddress"`
				EStatus            int    `json:"eStatus"`
				EMode              int    `json:"eMode"`
				ENeType            int    `json:"eNeType"`
			} `json:"RADIO_LINK_TERMINAL"`
			CtActive struct {
				BActualOutputPower       string `json:"bActualOutputPower"`
				BDescription             string `json:"bDescription"`
				BActualXpi               string `json:"bActualXpi"`
				BActualInputPower        string `json:"bActualInputPower"`
				BDistinguishedName       string `json:"bDistinguishedName"`
				LTxFrequency             int    `json:"lTxFrequency"`
				ETxOperStatus            int    `json:"eTxOperStatus"`
				ETxAdminStatus           int    `json:"eTxAdminStatus"`
				WFrameID                 int    `json:"wFrameId"`
				SbSelectedMinOutputPower int    `json:"sbSelectedMinOutputPower"`
				SbSelectedMaxOutputPower int    `json:"sbSelectedMaxOutputPower"`
				EStatus                  int    `json:"eStatus"`
				EXpicStatus              int    `json:"eXpicStatus"`
				ECarrierID               int    `json:"eCarrierId"`
				LActualTxCapacity        int    `json:"lActualTxCapacity"`
				EActualTxAcm             int    `json:"eActualTxAcm"`
			} `json:"CT_ACTIVE"`
			VirtualNode struct {
				EEquipmentProtection int `json:"eEquipmentProtection"`
				ESysController1State int `json:"eSysController1State"`
				ESysController2State int `json:"eSysController2State"`
				BActiveSlot          int `json:"bActiveSlot"`
				BoIsVirtualNode      int `json:"boIsVirtualNode"`
				EMode                int `json:"eMode"`
			} `json:"VIRTUAL_NODE"`
			RlwanPort struct {
				EMlhcAdminStatus int `json:"eMlhcAdminStatus"`
				EMlhcOperStatus  int `json:"eMlhcOperStatus"`
				EPlcAdminStatus  int `json:"ePlcAdminStatus"`
				EPlcOperStatus   int `json:"ePlcOperStatus"`
			} `json:"RLWAN_PORT"`
		} `json:"FE_STATUS_VIEW"`
	}

	info := &feRlInfo{}
	err = json.Unmarshal(body, info)
	if err != nil {
		return nil, fmt.Errorf("unmarshal fe rl info failed: %s", err)
	}

	if info == nil {
		return nil, fmt.Errorf("no fe rl info")
	}

	ip := ipconv.IntToIPv4(uint32(info.FeStatusView.RadioLinkTerminal.INeIPAddress))

	// Reverse ip slice
	for i, j := 0, len(ip)-1; i < j; i, j = i+1, j-1 {
		ip[i], ip[j] = ip[j], ip[i]
	}

	out := map[int]*map[string]string{
		0: {
			"sysName":    info.FeStatusView.RadioLinkTerminal.BNeName,
			"powerOut":   info.FeStatusView.CtActive.BActualOutputPower,
			"powerIn":    info.FeStatusView.CtActive.BActualInputPower,
			"txCapacity": fmt.Sprintf("%d", info.FeStatusView.CtActive.LActualTxCapacity*1000),
			"ip":         ip.String(),
		},
	}

	return out, err
}

// Get Software version
func (sd *deviceEricssonMlPt) SwVersion() (string, error) {
	sw := "Na"
	body, err := sd.WebApiGet("CATEGORY=JSONREQUEST&SOFTWARE")
	if err != nil {
		return sw, fmt.Errorf("get request from device api failed: %s", err)
	}

	type swInfo struct {
		Software struct {
			BRunningNR       string `json:"bRunningNR"`
			BRunningRelease  string `json:"bRunningRelease"`
			BRollbackNR      string `json:"bRollbackNR"`
			BRollbackRelease string `json:"bRollbackRelease"`
			TActivationTime  int    `json:"tActivationTime"`
			TDownloadTime    int    `json:"tDownloadTime"`
			EStatus          int    `json:"eStatus"`
			TStatusTimestamp int    `json:"tStatusTimestamp"`
			BProgress        int    `json:"bProgress"`
			BLastLogEntry    int    `json:"bLastLogEntry"`
		} `json:"SOFTWARE"`
	}

	info := &swInfo{}
	err = json.Unmarshal(body, info)
	if err != nil {
		return sw, fmt.Errorf("unmarshal software info failed: %s", err)
	}

	if info == nil {
		return sw, fmt.Errorf("no software info")
	}

	sw = strings.TrimSuffix(info.Software.BRunningNR, ".def")
	return sw, err
}

// Get last backup info
func (sd *deviceEricssonMlPt) LastBackup() (backupInfo, error) {
	out := backupInfo{}
	if err := sd.WebAuth(sd.webSession.cred); err != nil {
		return out, fmt.Errorf("error: WebAuth - %s", err)
	}

	body, err := sd.WebApiGet("CATEGORY=JSONREQUEST&CDB")
	if err != nil {
		return out, fmt.Errorf("get request from device api failed: %s", err)
	}

	err = sd.WebLogout()
	if err != nil {
		return out, fmt.Errorf("errors: WebLogout - %s", err)
	}

	// Last backup info provided by MINI-LINK PT web API
	type lastBackup struct {
		Cdb struct {
			I6LastRestoreServerIPV6 string `json:"i6LastRestoreServerIPV6"`
			BLastBackUpFile         string `json:"bLastBackUpFile"`
			I6LastBackUpServerIPV6  string `json:"i6LastBackUpServerIPV6"`
			TLastBackUpTime         int    `json:"tLastBackUpTime"`
			ILastBackUpServer       int    `json:"iLastBackUpServer"`
			TLastRestoreTime        int    `json:"tLastRestoreTime"`
			TLastChangeTime         int    `json:"tLastChangeTime"`
			EStatus                 int    `json:"eStatus"`
			TStatusTimestamp        int    `json:"tStatusTimestamp"`
			BProgress               int    `json:"bProgress"`
			EAutomaticRollback      int    `json:"eAutomaticRollback"`
			TPendingRollback        int    `json:"tPendingRollback"`
		} `json:"CDB"`
	}

	info := &lastBackup{}
	err = json.Unmarshal(body, info)
	if err != nil {
		return out, fmt.Errorf("unmarshal backup info failed: %s", err)
	}

	if info == nil {
		return out, fmt.Errorf("no backup info")
	}

	ip := ipconv.IntToIPv4(uint32(info.Cdb.ILastBackUpServer))

	// Reverse ip slice
	for i, j := 0, len(ip)-1; i < j; i, j = i+1, j-1 {
		ip[i], ip[j] = ip[j], ip[i]
	}

	if info.Cdb.TLastBackUpTime > 0 {
		out.TargetIP = ip.String()
		out.TargetFile = info.Cdb.BLastBackUpFile
		out.Timestamp = info.Cdb.TLastBackUpTime
		out.Progress = info.Cdb.BProgress
		if info.Cdb.EStatus == 7 {
			out.Success = true
		}
	}

	return out, err
}
