package godevman

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Adds Mikrotik specific functionality to snmpCommon type
type deviceMikrotik struct {
	snmpCommon
}

// Get running software version
func (sd *deviceMikrotik) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.14988.1.1.4.4.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Mobile modem signal data
func (sd *deviceMikrotik) MobSignal() (map[string]MobSignal, error) {
	ret := make(map[string]MobSignal)
	oid := ".1.3.6.1.4.1.14988.1.1.16.1.1"
	r, err := sd.getmulti(oid, nil)
	if err != nil {
		return nil, err
	}

	reIdxs := regexp.MustCompile(`\.(\d+)$`)
	tech := map[int64]string{
		-1: "unknown",
		0:  "gsmcompact",
		1:  "gsm",
		2:  "utran",
		3:  "egprs",
		4:  "hsdpa",
		5:  "hsupa",
		6:  "hsdpahsupa",
		7:  "eutran",
	}

	for o, d := range r {
		parts := reIdxs.FindStringSubmatch(string(o))
		ifIdx := parts[1]
		out, ok := ret[ifIdx]
		if !ok {
			out = MobSignal{}
		}
		switch o {
		case oid + ".11." + ifIdx:
			out.Imei.IsSet = true
			out.Imei.String = strings.TrimSpace(d.OctetString)
		case oid + ".2." + ifIdx:
			v := SensorVal{
				Unit:    "dBm",
				Divisor: 1,
				Value:   IntAbs(d.Integer),
				IsSet:   true,
			}
			if d.Integer < 0 {
				v.Divisor = -1
			}
			out.Rssi = v
		case oid + ".3." + ifIdx:
			v := SensorVal{
				Unit:    "dB",
				Divisor: 1,
				Value:   IntAbs(d.Integer),
				IsSet:   true,
			}
			if d.Integer < 0 {
				v.Divisor = -1
			}
			out.Rsrq = v
		case oid + ".4." + ifIdx:
			v := SensorVal{
				Unit:    "dBm",
				Divisor: 1,
				Value:   IntAbs(d.Integer),
				IsSet:   true,
			}
			if d.Integer < 0 {
				v.Divisor = -1
			}
			out.Rsrp = v
		case oid + ".5." + ifIdx:
			v := SensorVal{
				Unit:    "",
				Divisor: 1,
				Value:   IntAbs(d.Integer),
				IsSet:   true,
			}
			if d.Integer < 0 {
				v.Divisor = -1
			}
			out.CellId = v
		case oid + ".7." + ifIdx:
			v := SensorVal{
				Unit:    "dB",
				Divisor: 1,
				Value:   IntAbs(d.Integer),
				IsSet:   true,
			}
			if d.Integer < 0 {
				v.Divisor = -1
			}
			out.Sinr = v
		case oid + ".6." + ifIdx:
			if v, ok := tech[d.Integer]; ok {
				out.Technology.IsSet = true
				out.Technology.String = v
			}
		}
		ret[ifIdx] = out
	}

	return ret, nil
}

// Prepare CLI session parameters
func (sd *deviceMikrotik) cliPrepare() (*CliParams, error) {
	defParams, err := sd.snmpCommon.cliPrepare()
	if err != nil {
		return nil, err
	}

	params := defParams

	// make device specific changes to default parameters
	params.Cred[0] = params.Cred[0] + "+ct600w"
	if sd.cliSession.params.PromptRe == "" {
		params.PromptRe = `\] (\/.+)?>\s+$`
	}
	if sd.cliSession.params.ErrRe == "" {
		params.ErrRe = `(?im)(failure|error|unknown|unrecognized|invalid|not recognized|examples:|bad command)`
	}
	if sd.cliSession.params.DisconnectCmds == nil {
		params.DisconnectCmds = []string{"/quit"}
	}

	return params, nil
}

// Execute cli commands
func (sd *deviceMikrotik) RunCmds(c []string, o *CliCmdOpts) ([]string, error) {
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

	out, err := sd.cliCmds(c, o.ChkErr)
	if err != nil {
		return out, err
	}

	err = sd.closeCli()
	if err != nil {
		return out, err
	}

	return out, nil
}

// Get info from CLI
// Returns vlan id-s and names
func (sd *deviceMikrotik) D1qVlans() (map[string]string, error) {
	var out = make(map[string]string)

	info, err := sd.vlanInfo()
	if err != nil {
		return out, err
	}

	for k, v := range info {
		// set vlan name to empty string if not present
		name := func() string {
			for _, i := range v {
				if c, ok := i["comment"]; ok {
					return c
				}
			}
			return ""
		}()

		out[k] = name
	}

	return out, nil
}

// Get info from CLI
// Returns vlan info
func (sd *deviceMikrotik) D1qVlanInfo() (map[string]*D1qVlanInfo, error) {
	var out = make(map[string]*D1qVlanInfo)

	r, err := sd.IfInfo([]string{"Descr"})
	if err != nil {
		return out, fmt.Errorf("ifinfo error: %v", err)
	}

	var ifdescrIndex = make(map[string]int)
	for k, v := range r {
		if !v.Descr.IsSet {
			continue
		}

		i, err := strconv.Atoi(k)
		if err != nil {
			return out, fmt.Errorf("ifIdx Atoi error: %v", err)
		}

		ifdescrIndex[v.Descr.Value] = i
	}

	info, err := sd.vlanInfo()
	if err != nil {
		return out, err
	}

	for vlan, data := range info {
		vi := &D1qVlanInfo{
			Ports: make(map[int]*D1qVlanBrPort),
		}

		// parameter present check
		paramValue := func(s string, m map[string]string) string {
			if c, ok := m[s]; ok {
				return c
			}
			return ""
		}

		for _, i := range data {
			if vi.Name == "" {
				vi.Name = paramValue("name", i)
			}

			com := paramValue("comment", i)
			if com != "" {
				if vi.Name == "" {
					vi.Name = com
				} else {
					vi.Name = vi.Name + "; " + com
				}
			}

			if paramValue("interface", i) != "" {
				ports := strings.Split(paramValue("interface", i), ",")
				for _, port := range ports {
					if ifIdx, ok := ifdescrIndex[port]; ok {
						if _, ok := vi.Ports[ifIdx]; !ok {
							vi.Ports[ifIdx] = &D1qVlanBrPort{
								IfIdx: ifIdx,
							}
						}

						p := vi.Ports[ifIdx]
						if paramValue("use-service-tag", i) == "no" {
							p.UnTag = true
						}
					}
				}
			} else if paramValue("ports", i) != "" {
				ports := strings.Split(paramValue("ports", i), ",")
				for _, port := range ports {
					if ifIdx, ok := ifdescrIndex[port]; ok {
						if _, ok := vi.Ports[ifIdx]; !ok {
							vi.Ports[ifIdx] = &D1qVlanBrPort{
								IfIdx: ifIdx,
								UnTag: true,
							}
						}
					}
				}
			} else if paramValue("tagged", i) != "" || paramValue("untagged", i) != "" {
				ports := strings.Split(paramValue("tagged", i), ",")
				for _, port := range ports {
					if ifIdx, ok := ifdescrIndex[port]; ok {
						if _, ok := vi.Ports[ifIdx]; !ok {
							vi.Ports[ifIdx] = &D1qVlanBrPort{
								IfIdx: ifIdx,
							}
						}
					}
				}
				ports = strings.Split(paramValue("untagged", i), ",")
				for _, port := range ports {
					if ifIdx, ok := ifdescrIndex[port]; ok {
						if _, ok := vi.Ports[ifIdx]; !ok {
							vi.Ports[ifIdx] = &D1qVlanBrPort{
								IfIdx: ifIdx,
								UnTag: true,
							}
						}
					}
				}
			} else if paramValue("tagged-ports", i) != "" {
				ports := strings.Split(paramValue("tagged-ports", i), ",")
				for _, port := range ports {
					if ifIdx, ok := ifdescrIndex[port]; ok {
						if _, ok := vi.Ports[ifIdx]; ok {
							vi.Ports[ifIdx].UnTag = false
						}
					}
				}
			}
		}

		out[vlan] = vi
	}

	return out, nil
}

// Set via CLI
// Set Ethernet Interface Alias
// set - map of ifIndexes and related ifAliases
func (sd *deviceMikrotik) SetIfAlias(set map[string]string) error {
	idxs := make([]string, 0, len(set))
	for k := range set {
		idxs = append(idxs, k)
	}

	r, err := sd.IfInfo([]string{"Descr", "Alias"}, idxs...)
	if err != nil {
		return fmt.Errorf("ifinfo error: %v", err)
	}

	cmds := []string{"/interface ethernet"}

	for k, v := range set {
		if i, ok := r[k]; ok {
			if i.Alias.IsSet && i.Alias.Value != v && i.Descr.IsSet {
				cmd := "set [ find name=" + i.Descr.Value + "] comment=\"" + v + "\""
				cmds = append(cmds, cmd)
			}
		}
	}
	cmds = append(cmds, "/quit")

	_, err = sd.RunCmds(cmds, &CliCmdOpts{ChkErr: true})
	if err != nil {
		return fmt.Errorf("cli command error: %v", err)
	}

	return nil
}

// Get info from CLI
// Returns vlan info
func (sd *deviceMikrotik) vlanInfo() (map[string][]map[string]string, error) {
	var vlans = make(map[string][]map[string]string)

	cmds := []string{
		"/interface vlan print detail terse",
		"/interface ethernet switch vlan print terse detail",
		"/interface bridge vlan print terse detail",
		"/interface ethernet switch egress-vlan-tag print terse detail",
		"/quit",
	}

	r, err := sd.RunCmds(cmds, nil)
	if err != nil {
		return vlans, fmt.Errorf("cli command error: %v", err)
	}

	var rows []string
	for i, s := range r {
		if i%2 == 0 {
			continue
		}
		rows = append(rows, SplitLineEnd(s)...)
	}

	for _, row := range rows {
		if !strings.Contains(row, "vlan-id=") && !strings.Contains(row, "vlan-ids=") {
			continue
		}

		params := sd.terseParser(row)

		if vlan, ok := params["vlan-id"]; ok {
			vlans[vlan] = append(vlans[vlan], params)
		}
		if vlan, ok := params["vlan-ids"]; ok {
			vlans[vlan] = append(vlans[vlan], params)
		}
	}

	return vlans, nil
}

// Returns parameter-value map from row of `print terse detail` cli output
func (sd *deviceMikrotik) terseParser(row string) map[string]string {
	var out = make(map[string]string)

	parts := strings.Fields(row)

	for _, p := range parts {
		if !strings.Contains(p, "=") {
			continue
		}

		param := strings.Split(p, "=")
		out[param[0]] = param[1]
	}

	return out
}
