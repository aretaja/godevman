package godevman

import "strconv"

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

	// Get slice of ifIndexes from ipInfo and fill output map with ip info
	ifIdxs := make([]string, 0, len(ipInfo))
	for i, v := range ipInfo {
		ifIdxs = append(ifIdxs, strconv.FormatInt(int64(v.IfIdx), 10))

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
