package godevman

import (
	"fmt"
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

// Prepare CLI session parameters
func (sd *deviceMikrotik) cliPrepare() (*CliParams, error) {
	defParams, err := sd.snmpCommon.cliPrepare()
	if err != nil {
		return nil, err
	}

	params := defParams

	// make device specific changes to default parameters
	params.Cred[0] = params.Cred[0] + "+ct600w"
	if params.PromptRe == "" {
		params.PromptRe = `\] (\/.+)?>\s+$`
	}
	if params.ErrRe == "" {
		params.ErrRe = `(?im)(failure|error|unknown|unrecognized|invalid|not recognized|examples:|bad command)`
	}

	return params, nil
}

// Execute cli commands
func (sd *deviceMikrotik) RunCmds(c []string) ([]string, error) {
	p, err := sd.cliPrepare()
	if err != nil {
		return nil, err
	}

	err = sd.startCli(p)
	if err != nil {
		return nil, err
	}

	out, err := sd.cliCmds(c)
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
func (sd *deviceMikrotik) D1qVlanInfo() (map[string]*d1qVlanInfo, error) {
	var out = make(map[string]*d1qVlanInfo)

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
		vi := &d1qVlanInfo{
			Ports: make(map[int]*d1qVlanBrPort),
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
							vi.Ports[ifIdx] = &d1qVlanBrPort{
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
							vi.Ports[ifIdx] = &d1qVlanBrPort{
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
							vi.Ports[ifIdx] = &d1qVlanBrPort{
								IfIdx: ifIdx,
							}
						}
					}
				}
				ports = strings.Split(paramValue("untagged", i), ",")
				for _, port := range ports {
					if ifIdx, ok := ifdescrIndex[port]; ok {
						if _, ok := vi.Ports[ifIdx]; !ok {
							vi.Ports[ifIdx] = &d1qVlanBrPort{
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

	_, err = sd.RunCmds(cmds)
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

	r, err := sd.RunCmds(cmds)
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
