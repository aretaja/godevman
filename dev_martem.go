package godevman

import (
	"fmt"
	"regexp"
	"strings"
)

// Adds Martem specific SNMP functionality to snmpCommon type
type deviceMartem struct {
	snmpCommon
}

// Get running software version
func (sd *deviceMartem) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.43098.2.1.4.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Get device model
// Example output:
//  map[string]string{
// 	 "hwtype":"GWM-C1-N-M2",
// 	 "prodname":"Telem-GWM",
// 	 "serial":"GWM-2767"
// }
func (sd *deviceMartem) HwInfo() (map[string]string, error) {
	out := make(map[string]string)
	oid := ".1.3.6.1.4.1.43098.2.1"
	r, err := sd.getmulti(oid, []string{"6.0", "7.0", "8.0"})
	if err != nil {
		return out, err
	}

	for o, d := range r {
		switch o {
		case oid + ".7.0":
			out["serial"] = d.OctetString
		case oid + ".6.0":
			out["prodname"] = d.OctetString
		case oid + ".8.0":
			out["hwtype"] = d.OctetString
		}
	}

	return out, nil
}

// Mobile modem signal data
func (sd *deviceMartem) MobSignal() (map[string]MobSignal, error) {
	ret := make(map[string]MobSignal)
	oid := ".1.3.6.1.4.1.43098.2.4"
	r, err := sd.getmulti(oid, nil)
	if err != nil {
		return nil, err
	}

	reIdxs := regexp.MustCompile(`\.(\d+)$`)

	for o, d := range r {
		parts := reIdxs.FindStringSubmatch(string(o))
		ifIdx := parts[1]
		out, ok := ret[ifIdx]
		if !ok {
			out = MobSignal{}
		}

		switch o {
		case oid + ".1." + ifIdx:
			out.Registration.IsSet = true
			out.Registration.String = strings.TrimSpace(d.OctetString)
		case oid + ".2." + ifIdx:
			out.Technology.IsSet = true
			out.Technology.String = strings.TrimSpace(d.OctetString)
		case oid + ".3." + ifIdx:
			out.Band.IsSet = true
			out.Band.String = strings.TrimSpace(d.OctetString)
		case oid + ".4." + ifIdx:
			out.Operator.IsSet = true
			out.Operator.String = strings.TrimSpace(d.OctetString)
		case oid + ".5." + ifIdx:
			out.Ber.IsSet = true
			out.Ber.String = strings.TrimSpace(d.OctetString)
		case oid + ".7." + ifIdx:
			out.Imei.IsSet = true
			out.Imei.String = strings.TrimSpace(d.OctetString)
		case oid + ".8." + ifIdx:
			v := SensorVal{
				Unit:    "level(0-4)",
				Divisor: 1,
				Value:   uint64(d.Integer),
				IsSet:   true,
			}
			out.SignalBars = v
		}
		ret[ifIdx] = out
	}

	return ret, nil
}

// Execute cli commands
func (sd *deviceMartem) RunCmds(c []string, o *CliCmdOpts) ([]string, error) {
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
