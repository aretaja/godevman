package godevman

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Adds Teltonika specific functionality to snmpCommon type
type deviceTeltonika struct {
	snmpCommon
}

// Get running software version
func (sd *deviceTeltonika) SwVersion() (string, error) {
	oid := ".1.3.6.1.4.1.48690.1.6.0"
	r, err := sd.getone(oid)
	return r[oid].OctetString, err
}

// Get device model
// Example output:
//  map[string]string{
// 	 "hwtype":"0808",
// 	 "prodname":"RUTX0900XXXX",
// 	 "serial":"1113589271"
// }
func (sd *deviceTeltonika) HwInfo() (map[string]string, error) {
	out := make(map[string]string)
	oid := ".1.3.6.1.4.1.48690.1"
	r, err := sd.getmulti(oid, []string{"1.0", "3.0", "5.0"})
	if err != nil {
		return out, err
	}

	for o, d := range r {
		switch o {
		case oid + ".1.0":
			out["serial"] = d.OctetString
		case oid + ".3.0":
			out["prodname"] = d.OctetString
		case oid + ".5.0":
			out["hwtype"] = d.OctetString
		}
	}

	return out, nil
}

// Mobile modem signal data
func (sd *deviceTeltonika) MobSignal() (map[string]mobSignal, error) {
	ret := make(map[string]mobSignal)
	oid := ".1.3.6.1.4.1.48690.2.2.1"
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
			out = mobSignal{}
		}
		switch o {
		case oid + ".11." + ifIdx:
			out.Registration.IsSet = true
			out.Registration.String = strings.TrimSpace(d.OctetString)
		case oid + ".16." + ifIdx:
			out.Technology.IsSet = true
			out.Technology.String = strings.TrimSpace(d.OctetString)
		case oid + ".13." + ifIdx:
			out.Operator.IsSet = true
			out.Operator.String = strings.TrimSpace(d.OctetString)
		case oid + ".3." + ifIdx:
			out.Imei.IsSet = true
			out.Imei.String = strings.TrimSpace(d.OctetString)
		case oid + ".12." + ifIdx:
			v := sensorVal{
				Unit:    "dBm",
				Divisor: 1,
				Value:   IntAbs(d.Integer),
				IsSet:   true,
			}
			if d.Integer < 0 {
				v.Divisor = -1
			}
			out.Signal = v
		case oid + ".19." + ifIdx:
			f, _ := strconv.ParseFloat(strings.TrimSpace(d.OctetString), 64)
			v := sensorVal{
				Unit:    "dBm",
				Divisor: 10,
				Value:   uint64(math.Abs(f * 10)),
				IsSet:   true,
			}
			if f < 0 {
				v.Divisor = -10
			}
			out.Sinr = v
		case oid + ".20." + ifIdx:
			f, _ := strconv.ParseFloat(strings.TrimSpace(d.OctetString), 64)
			v := sensorVal{
				Unit:    "dBm",
				Divisor: 1,
				Value:   uint64(math.Abs(f)),
				IsSet:   true,
			}
			if f < 0 {
				v.Divisor = -1
			}
			out.Rsrp = v
		case oid + ".21." + ifIdx:
			f, _ := strconv.ParseFloat(strings.TrimSpace(d.OctetString), 64)
			v := sensorVal{
				Unit:    "dBm",
				Divisor: 10,
				Value:   uint64(math.Abs(f * 10)),
				IsSet:   true,
			}
			if f < 0 {
				v.Divisor = -10
			}
			out.Rsrq = v
		}
		ret[ifIdx] = out
	}

	return ret, nil
}
