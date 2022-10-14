package godevman

import (
	"fmt"
	"regexp"
)

// Adds Cisco specific SNMP functionality to snmpCommon type
type deviceCisco struct {
	snmpCommon
}

// Get running software version
func (sd *deviceCisco) SwVersion() (string, error) {
	var out string
	res, err := sd.System([]string{"Descr"})
	if err != nil {
		return out, err
	}

	re := regexp.MustCompile(`Version (.*?)[,|\s]`)
	if res.Descr.IsSet {
		reMatch := re.FindStringSubmatch(res.Descr.Value)
		if reMatch == nil {
			return out, fmt.Errorf("failed to parse sysDescr for version - %s", res.Descr.Value)
		}
		out = reMatch[1]
	}

	return out, nil
}

// Get Phase sync info
func (sd *deviceCisco) PhaseSyncInfo() (*PhaseSyncInfo, error) {
	var gbaseoids = map[string]string{
		"parentGm": ".1.3.6.1.4.1.9.9.760.1.2.2.1.8",
		"state":    ".1.3.6.1.4.1.9.9.760.1.2.4.1.4",
		"hops":     ".1.3.6.1.4.1.9.9.760.1.2.1.1.4",
	}

	var wbaseoids = map[string]string{
		"pNames": ".1.3.6.1.4.1.9.9.760.1.2.7.1.5",
		"pRoles": ".1.3.6.1.4.1.9.9.760.1.2.7.1.6",
	}

	var clockStateType = map[int64]string{
		1: "freerun",
		2: "holdover",
		3: "acquiring",
		4: "frequencyLocked",
		5: "phaseAligned",
	}

	var clockRoleType = map[int64]string{
		1: "master",
		2: "slave",
	}

	var out = new(PhaseSyncInfo)
	r, err := sd.snmpSession.Walk(gbaseoids["hops"], true, true)
	if err != nil && sd.handleErr(gbaseoids["hops"], err) {
		return out, err
	}

	if len(r) != 1 {
		return out, fmt.Errorf("multiple indexes not supported")
	}

	var idx string
	for idx = range r {
		break
	}

	out.HopsToGm = ValU64{
		Value: r[idx].Counter32,
		IsSet: true,
	}

	var n = make(map[string]string)
	var oids []string
	for name, bo := range gbaseoids {
		if name == "hops" {
			continue
		}
		oid := bo + "." + idx
		oids = append(oids, oid)
		n[oid] = name
	}

	r, err = sd.getmulti("", oids)
	if err != nil {
		return out, err
	}

	for o, res := range r {
		if n[o] == "parentGm" && res.Vtype == "OctetString" {
			out.ParentGmIdent = ValString{
				Value: res.OctetString,
				IsSet: true,
			}
		} else if n[o] == "state" && res.Vtype == "Integer" {
			if value, ok := clockStateType[res.Integer]; ok {
				out.State = ValString{
					Value: value,
					IsSet: true,
				}
			}
		}
	}

	pNames, err := sd.snmpSession.Walk(wbaseoids["pNames"]+"."+idx, true, true)
	if err != nil && sd.handleErr(wbaseoids["pNames"]+"."+idx, err) {
		return out, err
	}

	pRoles, err := sd.snmpSession.Walk(wbaseoids["pRoles"]+"."+idx, true, true)
	if err != nil && sd.handleErr(wbaseoids["pRoles"]+"."+idx, err) {
		return out, err
	}

	ports := make(map[string]string)
	for n, res := range pNames {
		name := fmt.Sprintf("%s(%s)", n, res.OctetString)
		state := ""
		if value, ok := clockRoleType[pRoles[n].Integer]; ok {
			state = value
		}
		ports[name] = state
	}

	out.PortsRole = ports

	return out, nil
}

// Prepare CLI session parameters
func (sd *deviceCisco) cliPrepare() (*CliParams, error) {
	defParams, err := sd.snmpCommon.cliPrepare()
	if err != nil {
		return nil, err
	}

	params := defParams

	// make device specific changes to default parameters
	if sd.cliSession.params.DisconnectCmds == nil {
		params.DisconnectCmds = []string{"end", "exit"}
	}
	if sd.cliSession.params.PreCmds == nil {
		params.PreCmds = []string{
			"terminal length 0",
			"terminal width 132",
		}
	}

	return params, nil
}

// Execute cli commands
func (sd *deviceCisco) RunCmds(c []string, o *CliCmdOpts) ([]string, error) {
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
