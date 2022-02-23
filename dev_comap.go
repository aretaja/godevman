package godevman

import (
	"fmt"
	"strings"
)

// Adds ComAp power generator specific functionality to snmpCommon type
type deviceComap struct {
	snmpCommon
}

// Get info from .iso.org.dod.internet.mgmt.mib-2.system and
// .iso.org.dod.internet.private.enterprises.enterprises-28634.il-14.groupRdCfg tree
// Replaces common snmp method
// Valid targets values: "All", "Descr", "ObjectID", "UpTime", "Contact", "Name", "Location"
func (sd *deviceComap) System(targets []string) (system, error) {
	var out system
	oids := map[string]string{
		"descr":    ".1.3.6.1.2.1.1.1.0",
		"objectID": ".1.3.6.1.2.1.1.2.0",
		"upTime":   ".1.3.6.1.2.1.1.3.0",
		"contact":  ".1.3.6.1.2.1.1.4.0",
		"name":     ".1.3.6.1.4.1.28634.14.4.8637.0",
		"location": ".1.3.6.1.2.1.1.6.0",
	}

	rOids := []string{}

	for _, t := range targets {
		switch t {
		case "All":
			rOids = []string{}
			for _, o := range oids {
				rOids = append(rOids, o)
			}
			continue
		case "Descr":
			rOids = append(rOids, oids["descr"])
		case "ObjectID":
			rOids = append(rOids, oids["objectID"])
		case "UpTime":
			rOids = append(rOids, oids["upTime"])
		case "Contact":
			rOids = append(rOids, oids["contact"])
		case "Name":
			rOids = append(rOids, oids["name"])
		case "Location":
			rOids = append(rOids, oids["location"])
		}
	}

	r, err := sd.snmpSession.Get(rOids)
	if err != nil {
		return out, err
	}

	for o, d := range r {
		switch o {
		case oids["descr"]:
			out.Descr.Value = d.OctetString
			out.Descr.IsSet = true
		case oids["objectID"]:
			out.ObjectID.Value = d.ObjectIdentifier
			out.ObjectID.IsSet = true
		case oids["upTime"]:
			out.UpTime.Value = d.TimeTicks
			out.UpTime.IsSet = true
			dt := UpTimeString(out.UpTime.Value, 0)
			out.UpTimeStr.Value = dt
			out.UpTimeStr.IsSet = true
		case oids["contact"]:
			out.Contact.Value = d.OctetString
			out.Contact.IsSet = true
		case oids["name"]:
			out.Name.Value = d.OctetString
			out.Name.IsSet = true
		case oids["location"]:
			out.Location.Value = d.OctetString
			out.Location.IsSet = true
		}
	}
	return out, err
}

// Get info from .iso.org.dod.internet.private.enterprises.enterprises-28634.il-14.groupRdCfg tree
// Valid targets values: "All", "Electrical", "Engine", "Common"
func (sd *deviceComap) GeneratorInfo(targets []string) (genInfo, error) {
	out := genInfo{}
	oid := ".1.3.6.1.4.1.28634.14.2"
	idxs := map[string]string{
		"genVoltL1":    "8192.0",
		"genVoltL2":    "8193.0",
		"genVoltL3":    "8194.0",
		"mainsVoltL1":  "8195.0",
		"mainsVoltL2":  "8196.0",
		"mainsVoltL3":  "8197.0",
		"genCurrentL1": "8198.0",
		"genCurrentL2": "8199.0",
		"genCurrentL3": "8200.0",
		"genPower":     "8202.0",
		"genFreq":      "8210.0",
		"runHours":     "8206.0",
		"numStarts":    "8207.0",
		"batteryVolt":  "8213.0",
		"fuelLevel":    "9153.0",
		"fuelConsum":   "9040.0",
		"coolantTemp":  "9151.0",
		"engineState":  "9244.0",
		"breakerState": "9245.0",
		"genMode":      "9887.0",
	}

	rIdxs := []string{}

	for _, t := range targets {
		switch t {
		case "All":
			rIdxs = []string{}
			for _, i := range idxs {
				rIdxs = append(rIdxs, i)
			}
			continue
		case "Electrical":
			rIdxs = []string{
				idxs["genVoltL1"], idxs["genVoltL2"], idxs["genVoltL3"], idxs["mainsVoltL1"],
				idxs["mainsVoltL2"], idxs["mainsVoltL3"], idxs["genCurrentL1"], idxs["genCurrentL2"],
				idxs["genCurrentL3"], idxs["genPower"], idxs["genFreq"],
			}
		case "Engine":
			rIdxs = []string{
				idxs["runHours"], idxs["numStarts"], idxs["batteryVolt"], idxs["fuelLevel"],
				idxs["fuelConsum"], idxs["coolantTemp"],
			}
		case "Common":
			rIdxs = []string{idxs["engineState"], idxs["breakerState"], idxs["genMode"]}
		default:
			return out, fmt.Errorf("unknown target: %s", t)
		}
	}

	r, err := sd.getmulti(oid, rIdxs)
	if err != nil {
		return out, err
	}

	for o, d := range r {
		switch {
		case strings.HasSuffix(o, idxs["genVoltL1"]):
			out.GenVoltL1.Unit = "V"
			out.GenVoltL1.Value = uint64(d.Integer)
			out.GenVoltL1.IsSet = true
		case strings.HasSuffix(o, idxs["genVoltL2"]):
			out.GenVoltL2.Unit = "V"
			out.GenVoltL2.Value = uint64(d.Integer)
			out.GenVoltL2.IsSet = true
		case strings.HasSuffix(o, idxs["genVoltL3"]):
			out.GenVoltL3.Unit = "V"
			out.GenVoltL3.Value = uint64(d.Integer)
			out.GenVoltL3.IsSet = true
		case strings.HasSuffix(o, idxs["mainsVoltL1"]):
			out.MainsVoltL1.Unit = "V"
			out.MainsVoltL1.Value = uint64(d.Integer)
			out.MainsVoltL1.IsSet = true
		case strings.HasSuffix(o, idxs["mainsVoltL2"]):
			out.MainsVoltL2.Unit = "V"
			out.MainsVoltL2.Value = uint64(d.Integer)
			out.MainsVoltL2.IsSet = true
		case strings.HasSuffix(o, idxs["mainsVoltL3"]):
			out.MainsVoltL3.Unit = "V"
			out.MainsVoltL3.Value = uint64(d.Integer)
			out.MainsVoltL3.IsSet = true
		case strings.HasSuffix(o, idxs["genCurrentL1"]):
			out.GenCurrentL1.Unit = "A"
			out.GenCurrentL1.Value = uint64(d.Integer)
			out.GenCurrentL1.IsSet = true
		case strings.HasSuffix(o, idxs["genCurrentL2"]):
			out.GenCurrentL2.Unit = "A"
			out.GenCurrentL2.Value = uint64(d.Integer)
			out.GenCurrentL2.IsSet = true
		case strings.HasSuffix(o, idxs["genCurrentL3"]):
			out.GenCurrentL3.Unit = "A"
			out.GenCurrentL3.Value = uint64(d.Integer)
			out.GenCurrentL3.IsSet = true
		case strings.HasSuffix(o, idxs["genPower"]):
			out.GenPower.Unit = "kW"
			out.GenPower.Value = uint64(d.Integer)
			out.GenPower.IsSet = true
		case strings.HasSuffix(o, idxs["genFreq"]):
			out.GenFreq.Unit = "Hz"
			out.GenFreq.Divisor = 10
			out.GenFreq.Value = uint64(d.Integer)
			out.GenFreq.IsSet = true
		case strings.HasSuffix(o, idxs["runHours"]):
			out.RunHours.Unit = "h"
			out.RunHours.Divisor = 10
			out.RunHours.Value = uint64(d.Integer)
			out.RunHours.IsSet = true
		case strings.HasSuffix(o, idxs["numStarts"]):
			out.NumStarts.Value = uint64(d.Integer)
			out.NumStarts.IsSet = true
		case strings.HasSuffix(o, idxs["batteryVolt"]):
			out.BatteryVolt.Unit = "V"
			out.BatteryVolt.Divisor = 10
			out.BatteryVolt.Value = uint64(d.Integer)
			out.BatteryVolt.IsSet = true
		case strings.HasSuffix(o, idxs["fuelLevel"]):
			out.FuelLevel.Unit = "%"
			out.FuelLevel.Value = uint64(d.Integer)
			out.FuelLevel.IsSet = true
		case strings.HasSuffix(o, idxs["fuelConsum"]):
			out.FuelConsum.Unit = "l"
			out.FuelConsum.Value = uint64(d.Integer)
			out.FuelConsum.IsSet = true
		case strings.HasSuffix(o, idxs["coolantTemp"]):
			out.CoolantTemp.Unit = "°C"
			out.CoolantTemp.Value = IntAbs(d.Integer)
			out.CoolantTemp.IsSet = true
			if d.Integer < 0 {
				out.CoolantTemp.Divisor = -1
			}
		case strings.HasSuffix(o, idxs["engineState"]):
			states := map[int64]string{
				0:  "Init",
				1:  "Ready",
				2:  "NotReady",
				3:  "Prestart",
				4:  "Cranking",
				5:  "Pause",
				6:  "Starting",
				7:  "Running",
				8:  "Loaded",
				9:  "SoftUnld",
				10: "Cooling",
				11: "Stop",
				12: "Shutdown",
				13: "Ventil",
				14: "EmergMan",
				15: "SoftLoad",
				16: "WaitStop",
				17: "SDVentil",
			}
			if v, ok := states[d.Integer]; ok && d.Vtype == "Integer" {
				out.EngineState.Value = v
				out.EngineState.IsSet = true
			}
		case strings.HasSuffix(o, idxs["breakerState"]):
			states := map[int64]string{
				0:  "Init",
				1:  "BrksOff",
				2:  "IslOper",
				3:  "MainsOper",
				4:  "ParalOper",
				5:  "RevSync",
				6:  "Synchro",
				7:  "MainsFlt",
				8:  "ValidFlt",
				9:  "MainsRet",
				10: "MultIslOp",
				11: "MultParOp",
				12: "EmergMan",
			}
			if v, ok := states[d.Integer]; ok && d.Vtype == "Integer" {
				out.BreakerState.Value = v
				out.BreakerState.IsSet = true
			}
		case strings.HasSuffix(o, idxs["genMode"]):
			states := map[int64]string{
				0: "Off",
				1: "Man",
				2: "Auto",
				3: "Test",
			}
			if v, ok := states[d.Integer]; ok && d.Vtype == "Integer" {
				out.GenMode.Value = v
				out.GenMode.IsSet = true
			}
		}
	}

	return out, nil
}
