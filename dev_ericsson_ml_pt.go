package godevman

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	"github.com/praserx/ipconv"
	"golang.org/x/crypto/ssh"
)

// Adds Ericsson MINI-LINK PT specific SNMP functionality to snmpCommon type
type deviceEricssonMlPt struct {
	snmpCommon
}

// Get IP Interface info
func (sd *deviceEricssonMlPt) IpIfInfo(ip ...string) (map[string]*ipIfInfo, error) {
	out := make(map[string]*ipIfInfo)

	ipInfo, err := sd.IpInfo(ip...)
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

// Get RL info (map keys are radio ifdescriptions)
func (sd *deviceEricssonMlPt) RlInfo() (map[string]*rlRadioIfInfo, error) {
	out := make(map[string]*rlRadioIfInfo)

	ctTable := ".1.3.6.1.4.1.193.223.2.7.1.1."
	rltTable := ".1.3.6.1.4.1.193.223.2.7.3.1."
	perfTable := ".1.3.6.1.4.1.193.223.2.7.4.1.1."
	tempOid := ".1.3.6.1.4.1.193.223.2.4.1.1.2.1"

	// Get CD and RLT ifdescr
	descrOids := []string{ctTable + "5", rltTable + "1"}

	ctDescr := make(map[string]string)
	rltDescr := make(map[string]string)
	for _, oid := range descrOids {
		r, err := sd.getmulti(oid, nil)
		if err != nil {
			return out, err
		}

		for o, d := range r {
			switch {
			case strings.Contains(o, ctTable+"5."):
				i := strings.TrimPrefix(o, ctTable+"5.")
				ctDescr[i] = d.OctetString
			case strings.Contains(o, rltTable+"1."):
				i := strings.TrimPrefix(o, rltTable+"1.")
				rltDescr[i] = d.OctetString
			}
		}
	}

	oids := []string{tempOid}
	for i := range ctDescr {
		oids = append(oids,
			ctTable+"1."+i, ctTable+"2."+i, ctTable+"6."+i, ctTable+"8."+i, ctTable+"25."+i,
			ctTable+"26."+i, ctTable+"43."+i, ctTable+"46."+i, perfTable+"4."+i, perfTable+"9."+i,
		)
	}

	for i := range rltDescr {
		oids = append(oids, rltTable+"7."+i)
	}

	r, err := sd.snmpSession.Get(oids)
	if err != nil {
		return out, err
	}

	rltStatus := map[int64]string{
		1: "Down",
		2: "Up",
		3: "Na",
		4: "Unkn",
		5: "Degraded",
	}

	opStatus := map[int64]string{
		1: "Unkn",
		2: "Off",
		3: "On",
		4: "Standby",
	}

	var temp valF64
	if v, ok := r[tempOid]; ok {
		t := strings.TrimSuffix(v.OctetString, " C")
		if s, err := strconv.ParseFloat(t, 64); err == nil {
			temp.Value = s
			temp.IsSet = true
		}
	}

	for idx, rd := range rltDescr {
		i := new(rlRadioIfInfo)
		i.Rau = make(map[string]*rauInfo)
		i.Descr.Value = rd
		i.Descr.IsSet = true
		if s, err := strconv.Atoi(idx); err == nil {
			i.IfIdx.Value = s
			i.IfIdx.IsSet = true
		}
		if v, ok := r[rltTable+"7."+idx]; ok {
			if s, ok := rltStatus[v.Integer]; ok {
				i.OperStat.Value = s
				i.OperStat.IsSet = true
			}
		}

		for cIdx, cd := range ctDescr {
			rf := new(rfInfo)
			rf.Descr.Value = cd
			rf.Descr.IsSet = true
			if s, err := strconv.Atoi(cIdx); err == nil {
				rf.IfIdx.Value = s
				rf.IfIdx.IsSet = true
			}

			rau := new(rauInfo)
			rau.Descr.Value = rd
			rau.Descr.IsSet = true
			rau.Temp = temp
			rau.Rf = make(map[string]*rfInfo)

			rau.Rf[cd] = rf
			if v, ok := r[ctTable+"1."+cIdx]; ok {
				if s, err := strconv.ParseFloat(v.OctetString, 64); err == nil {
					rf.PowerIn.Value = s
					rf.PowerIn.IsSet = true
				}
			}
			if v, ok := r[ctTable+"2."+cIdx]; ok {
				if s, err := strconv.ParseFloat(v.OctetString, 64); err == nil {
					rf.PowerOut.Value = s
					rf.PowerOut.IsSet = true
				}
			}
			if v, ok := r[ctTable+"6."+cIdx]; ok {
				rf.Name.Value = v.OctetString
				rf.Name.IsSet = true
			}
			if v, ok := r[ctTable+"25."+cIdx]; ok {
				if s, ok := opStatus[v.Integer]; ok {
					rf.Status.Value = s
					rf.Status.IsSet = true
				}
			}
			if v, ok := r[ctTable+"26."+cIdx]; ok {
				if v.Integer == 1 {
					rf.Mute.Value = true
				}
				rf.Mute.IsSet = true
			}
			if v, ok := r[ctTable+"43."+cIdx]; ok {
				if s, err := strconv.ParseFloat(v.OctetString, 64); err == nil {
					rf.Snr.Value = s
					rf.Snr.IsSet = true
				}
			}
			if v, ok := r[ctTable+"46."+cIdx]; ok {
				rf.TxCapacity.Value = int(v.Integer) * 1000
				rf.TxCapacity.IsSet = true
			}
			// Return error counters only if carrier is in up or degraded state
			if v, ok := r[ctTable+"8."+cIdx]; ok {
				if v.Integer == 2 || v.Integer == 5 {
					if cv, ok := r[perfTable+"4."+cIdx]; ok {
						i.Es.Value = int(cv.Integer)
						i.Es.IsSet = true
					}
					if cv, ok := r[perfTable+"9."+cIdx]; ok {
						i.Uas.Value = int(cv.Integer)
						i.Uas.IsSet = true
					}
				}
			}

			i.Rau[cd] = rau
		}

		out[rd] = i
	}

	return out, nil
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
func (sd *deviceEricssonMlPt) LastBackup() (*backupInfo, error) {
	if err := sd.WebAuth(sd.webSession.cred); err != nil {
		return nil, fmt.Errorf("error: WebAuth - %s", err)
	}

	body, err := sd.WebApiGet("CATEGORY=JSONREQUEST&CDB")
	if err != nil {
		return nil, fmt.Errorf("get request from device api failed: %s", err)
	}

	err = sd.WebLogout()
	if err != nil {
		return nil, fmt.Errorf("errors: WebLogout - %s", err)
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
		return nil, fmt.Errorf("unmarshal backup info failed: %s", err)
	}

	if info == nil {
		return nil, fmt.Errorf("no backup info")
	}

	ip := ipconv.IntToIPv4(uint32(info.Cdb.ILastBackUpServer))

	// Reverse ip slice
	for i, j := 0, len(ip)-1; i < j; i, j = i+1, j-1 {
		ip[i], ip[j] = ip[j], ip[i]
	}

	out := new(backupInfo)
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

// Get RL neighbour info (map keys are local ifdescriptions or "0" for PtP links)
func (sd *deviceEricssonMlPt) RlNbrInfo() (map[string]*rlRadioFeIfInfo, error) {
	res := new(rlRadioFeIfInfo)

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

	res.FeIfDescr = valString{
		Value: info.FeStatusView.RadioLinkTerminal.BDistinguishedName,
		IsSet: true,
	}
	res.SysName = valString{
		Value: info.FeStatusView.RadioLinkTerminal.BNeName,
		IsSet: true,
	}
	res.Ip = valString{
		Value: ip.String(),
		IsSet: true,
	}
	res.TxCapacity = valInt{
		Value: info.FeStatusView.CtActive.LActualTxCapacity * 1000,
		IsSet: true,
	}

	if v, err := strconv.ParseFloat(info.FeStatusView.CtActive.BActualInputPower, 64); err == nil {
		res.PowerIn.Value = v
		res.PowerIn.IsSet = true
	}

	if v, err := strconv.ParseFloat(info.FeStatusView.CtActive.BActualOutputPower, 64); err == nil {
		res.PowerOut.Value = v
		res.PowerOut.IsSet = true
	}

	out := map[string]*rlRadioFeIfInfo{"0": res}
	return out, err
}

// Prepare CLI session parameters
func (sd *deviceEricssonMlPt) cliPrepare() (*CliParams, error) {
	defParams, err := sd.snmpCommon.cliPrepare()
	if err != nil {
		return nil, err
	}

	params := defParams

	// make device specific changes to default parameters
	if params.PromptRe == "" {
		params.PromptRe = `[\)\]]#\s+$`
	}
	if params.ErrRe == "" {
		params.ErrRe = `(?im)(error|unknown|invalid|failed|timed out|no attribute)`
	}
	if params.DisconnectCmds == nil {
		params.DisconnectCmds = []string{"end;_", "exit"}
	}

	return params, nil
}

// Create and store cli expect client and update d.cliSession.params
func (d *deviceEricssonMlPt) startCli(p *CliParams) error {
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
	}

	// Regexes for credentials
	uRe := regexp.MustCompile(`login: ?$`)
	pRe := regexp.MustCompile(`password: ?$`)

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
func (sd *deviceEricssonMlPt) RunCmds(c []string, e bool) ([]string, error) {
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

// Execute cli commands
func (sd *deviceEricssonMlPt) DoBackup() error {
	if sd.backupParams == nil {
		return fmt.Errorf("device backup parameters are not defined")
	}

	// Get backup parameters
	host := sd.backupParams.TargetIp
	if host == "" {
		return fmt.Errorf("target ip is not defined")
	}
	user := ""
	if len(sd.backupParams.Cred) > 0 {
		user = sd.backupParams.Cred[0]
	}
	pass := ""
	if len(sd.backupParams.Cred) > 1 {
		pass = sd.backupParams.Cred[1]
	}

	t := time.Now()
	loc, _ := time.LoadLocation(sd.timeZone)
	t = t.In(loc)

	targetFile := sd.backupParams.BasePath + "/" + sd.backupParams.DevIdent + "_" + t.Format(time_iso8601_sec) + ".zip"

	cmds := []string{
		"config common cdb backup filename " + targetFile + " ip " + host + " mode sftp password " + pass + " port 22 user " + user + ";_",
		"quit",
	}

	res, err := sd.RunCmds(cmds, true)
	if err != nil {
		return fmt.Errorf("cli error: %v, output: %s", err, res)
	}

	for i := 0; i < 10; i++ {
		time.Sleep(3 * time.Second)
		b, err := sd.LastBackup()
		if err != nil {
			return fmt.Errorf("web api error: %v", err)
		}

		if t.Unix() < int64(b.Timestamp) && b.Progress == 100 && b.Success {
			return nil
		}
	}

	return fmt.Errorf("no confirm for backup success from web api")
}
