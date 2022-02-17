package godevman

import (
	"math"
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
func (sd *deviceTeltonika) MobSignal() (mobSignal, error) {
	out := new(mobSignal)
	oid := ".1.3.6.1.4.1.48690.2.2.1"
	r, err := sd.getmulti(oid, nil)
	if err != nil {
		return *out, err
	}

	for o, d := range r {
		switch o {
		case oid + ".11.1":
			out.Registration.IsSet = true
			out.Registration.String = strings.TrimSpace(d.OctetString)
		case oid + ".16.1":
			out.Technology.IsSet = true
			out.Technology.String = strings.TrimSpace(d.OctetString)
		case oid + ".13.1":
			out.Operator.IsSet = true
			out.Operator.String = strings.TrimSpace(d.OctetString)
		case oid + ".3.1":
			out.Imei.IsSet = true
			out.Imei.String = strings.TrimSpace(d.OctetString)
		case oid + ".12.1":
			v := sensorVal{
				Unit:    "dBm",
				Divisor: 1,
				Value:   uint64(math.Abs(float64(d.Integer))),
				IsSet:   true,
			}
			if d.Integer < 0 {
				v.Divisor = -1
			}
			out.Signal = v
		case oid + ".19.1":
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
		case oid + ".20.1":
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
		case oid + ".21.1":
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
	}

	return *out, nil
}
