package godevman

import (
	"fmt"
	"regexp"

	"github.com/aretaja/snmphelper"
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
		"pNames":  ".1.3.6.1.4.1.9.9.760.1.2.9.1.5",
		"pStates": ".1.3.6.1.4.1.9.9.760.1.2.9.1.6",
	}

	var clockStateType = map[int64]string{
		1: "freerun",
		2: "holdover",
		3: "acquiring",
		4: "frequencyLocked",
		5: "phaseAligned",
	}

	var clockPortState = map[int64]string{
		1: "initializing",
		2: "faulty",
		3: "disabled",
		4: "listening",
		5: "preMaster",
		6: "master",
		7: "passive",
		8: "uncalibrated",
		9: "slave",
	}
	var out = new(PhaseSyncInfo)
	r, err := sd.snmpSession.Walk(gbaseoids["hops"], true, true)
	if err != nil && sd.handleErr(gbaseoids["hops"], err) {
		return out, err
	}

	if len(r) > 1 {
		return out, fmt.Errorf("multiple indexes not supported")
	} else if len(r) == 0 {
		return out, fmt.Errorf("no indexes found")
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

	sNames, err := sd.snmpSession.Walk(wbaseoids["pNames"]+"."+idx, true, true)
	if err != nil && sd.handleErr(wbaseoids["pNames"]+"."+idx, err) {
		return out, err
	}

	pStates, err := sd.snmpSession.Walk(wbaseoids["pStates"]+"."+idx, true, true)
	if err != nil && sd.handleErr(wbaseoids["pStates"]+"."+idx, err) {
		return out, err
	}

	srcs := make(map[string]string)
	for n, res := range sNames {
		name := fmt.Sprintf("%s(%s)", n, res.OctetString)
		state := ""
		if value, ok := clockPortState[pStates[n].Integer]; ok {
			state = value
		}
		srcs[name] = state
	}

	out.SrcsState = srcs

	return out, nil
}

// Get Frequency sync info
func (sd *deviceCisco) FreqSyncInfo() (*FreqSyncInfo, error) {
	var baseoids = map[string]string{
		"cMode":    ".1.3.6.1.4.1.9.9.761.1.1.1.1.3",
		"qLevel":   ".1.3.6.1.4.1.9.9.761.1.1.2.1.4",
		"srcNames": ".1.3.6.1.4.1.9.9.761.1.1.3.1.2",
		"srcQ":     ".1.3.6.1.4.1.9.9.761.1.1.3.1.11",
	}

	var clockMode = map[int64]string{
		1: "unknown",
		2: "freerun",
		3: "holdover",
		4: "locked",
	}

	var qualityLevel = map[int64]string{
		1:  "NULL",
		2:  "DNU",
		3:  "DUS",
		4:  "FAILED",
		5:  "INV0",
		6:  "INV1",
		7:  "INV2",
		8:  "INV3",
		9:  "INV4",
		10: "INV5",
		11: "INV6",
		12: "INV7",
		13: "INV8",
		14: "INV9",
		15: "INV10",
		16: "INV11",
		17: "INV12",
		18: "INV13",
		19: "INV14",
		20: "INV15",
		21: "NSUPP",
		22: "PRC",
		23: "PROV",
		24: "PRS",
		25: "SEC",
		26: "SMC",
		27: "SSUA",
		28: "SSUB",
		29: "ST2",
		30: "ST3",
		31: "ST3E",
		32: "ST4",
		33: "STU",
		34: "TNC",
		35: "UNC",
		36: "UNK",
	}

	var out = new(FreqSyncInfo)
	var res = make(map[string]snmphelper.SnmpOut)
	for name, o := range baseoids {
		r, err := sd.snmpSession.Walk(o, true, true)
		if err != nil && sd.handleErr(o, err) {
			return out, err
		}
		res[name] = r
	}

	if len(res["cMode"]) != 1 && len(res["qLevel"]) != 1 {
		return out, fmt.Errorf("multiple indexes not supported")
	}

	var idx string
	for idx = range res["cMode"] {
		break
	}

	if res["cMode"][idx].Vtype == "Integer" {
		if value, ok := clockMode[res["cMode"][idx].Integer]; ok {
			out.ClockMode = ValString{
				Value: value,
				IsSet: true,
			}
		}
	}

	for idx = range res["qLevel"] {
		break
	}

	if res["qLevel"][idx].Vtype == "Integer" {
		if value, ok := qualityLevel[res["qLevel"][idx].Integer]; ok {
			out.ClockQaLevel = ValString{
				Value: value,
				IsSet: true,
			}
		}
	}

	srcs := make(map[string]string)
	for n, s := range res["srcNames"] {
		name := fmt.Sprintf("%s(%s)", n, s.OctetString)
		state := ""
		if value, ok := qualityLevel[res["srcQ"][n].Integer]; ok {
			state = value
		}
		srcs[name] = state
	}

	out.SrcsQaLevel = srcs

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
