package godevman

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/aretaja/snmphelper"
	"github.com/kr/pretty"
)

// Common SNMP functionality
type snmpCommon struct {
	device
}

// System

// Get info from .iso.org.dod.internet.mgmt.mib-2.system tree
// Valid targets values: "All", "Descr", "ObjectID", "UpTime", "Contact", "Name", "Location"
func (sd *snmpCommon) System(targets []string) (System, error) {
	var out System
	var idx []string

	for _, t := range targets {
		switch t {
		case "All":
			idx = []string{"1.0", "2.0", "3.0", "4.0", "5.0", "6.0"}
			continue
		case "Descr":
			idx = append(idx, "1.0")
		case "ObjectID":
			idx = append(idx, "2.0")
		case "UpTime":
			idx = append(idx, "3.0")
		case "Contact":
			idx = append(idx, "4.0")
		case "Name":
			idx = append(idx, "5.0")
		case "Location":
			idx = append(idx, "6.0")
		}
	}

	oid := ".1.3.6.1.2.1.1"
	r, err := sd.getmulti(oid, idx)
	if err != nil {
		return out, err
	}

	for o, d := range r {
		switch o {
		case oid + ".1.0":
			out.Descr.Value = d.OctetString
			out.Descr.IsSet = true
		case oid + ".2.0":
			out.ObjectID.Value = d.ObjectIdentifier
			out.ObjectID.IsSet = true
		case oid + ".3.0":
			out.UpTime.Value = d.TimeTicks
			// HACK some devices (fe. Ceragon IP50*) have errorly defined sysUpTime value as Gauge32
			if d.TimeTicks == 0 && d.Gauge32 != 0 {
				out.UpTime.Value = d.Gauge32
			}
			out.UpTime.IsSet = true
			if out.UpTime.IsSet {
				dt := UpTimeString(out.UpTime.Value, 0)
				out.UpTimeStr.Value = dt
				out.UpTimeStr.IsSet = true
			}
		case oid + ".4.0":
			out.Contact.Value = d.OctetString
			out.Contact.IsSet = true
		case oid + ".5.0":
			out.Name.Value = d.OctetString
			out.Name.IsSet = true
		case oid + ".6.0":
			out.Location.Value = d.OctetString
			out.Location.IsSet = true
		}
	}
	return out, err
}

// Get ifNumber
func (sd *snmpCommon) IfNumber() (int64, error) {
	oid := ".1.3.6.1.2.1.2.1.0"
	r, err := sd.getone(oid)
	return r[oid].Integer, err
}

// Get info from .iso.org.dod.internet.mgmt.mib-2.interfaces.ifTable and .iso.org.dod.internet.mgmt.mib-2.ifMIB.ifMIBObjects.ifXTable
// Valid targets values: "All", "AllNoIfx", "Descr", "Name", "Alias", "Type", "Mtu", "Speed", "Mac",
// "Admin", "Oper", "Last", "InOctets", "InUcast", "InMcast", "InBcast", "InDiscards", "InErrors",
// "OutOctets", "OutUcast", "OutMcast", "OutBcast", "OutDiscards", "OutErrors"
func (sd *snmpCommon) IfInfo(targets []string, idx ...string) (map[string]*IfInfo, error) {
	out := make(map[string]*IfInfo)
	iftable := ".1.3.6.1.2.1.2.2.1."
	ifxtable := ".1.3.6.1.2.1.31.1.1.1."
	var oids []string

	ifOids := []string{
		iftable + "2", iftable + "3", iftable + "4", iftable + "5", iftable + "6", iftable + "7",
		iftable + "8", iftable + "9", iftable + "10", iftable + "11", iftable + "13", iftable + "14",
		iftable + "16", iftable + "17",
	}

	ifxOids := []string{
		ifxtable + "1", ifxtable + "2", ifxtable + "3", ifxtable + "4", ifxtable + "5", ifxtable + "6",
		ifxtable + "7", ifxtable + "8", ifxtable + "9", ifxtable + "10", ifxtable + "11", ifxtable + "12",
		ifxtable + "13", ifxtable + "15", ifxtable + "18",
	}

	for _, t := range targets {
		switch t {
		case "All":
			oids = append(ifxOids, ifOids...)
			continue
		case "AllNoIfx":
			oids = ifOids
			continue
		case "Descr":
			oids = append(oids, iftable+"2")
		case "Name":
			oids = append(oids, ifxtable+"1")
		case "Alias":
			oids = append(oids, ifxtable+"18")
		case "Type":
			oids = append(oids, iftable+"3")
		case "Mtu":
			oids = append(oids, iftable+"4")
		case "Speed":
			oids = append(oids, ifxtable+"15")
			oids = append(oids, iftable+"5")
		case "Mac":
			oids = append(oids, iftable+"6")
		case "Admin":
			oids = append(oids, iftable+"7")
		case "Oper":
			oids = append(oids, iftable+"8")
		case "Last":
			oids = append(oids, iftable+"9")
		case "InOctets":
			oids = append(oids, ifxtable+"6")
			oids = append(oids, iftable+"10")
		case "InUcast":
			oids = append(oids, ifxtable+"7")
			oids = append(oids, iftable+"11")
		case "InMcast":
			oids = append(oids, ifxtable+"8")
		case "InBcast":
			oids = append(oids, ifxtable+"9")
		case "InDiscards":
			oids = append(oids, iftable+"13")
		case "InErrors":
			oids = append(oids, iftable+"14")
		case "OutOctets":
			oids = append(oids, ifxtable+"10")
			oids = append(oids, iftable+"16")
		case "OutUcast":
			oids = append(oids, ifxtable+"11")
			oids = append(oids, iftable+"17")
		case "OutMast":
			oids = append(oids, ifxtable+"12")
		case "OutBcast":
			oids = append(oids, ifxtable+"13")
		case "OutDiscards":
			oids = append(oids, iftable+"19")
		case "OutErrors":
			oids = append(oids, iftable+"20")
		}
	}

	mapEntry := func(oid, prefix string) string {
		i := strings.TrimPrefix(oid, prefix)
		if out[i] == nil {
			out[i] = new(IfInfo)
		}
		return i
	}

	// System UpTime
	sut, _ := sd.System([]string{"UpTime"})

	for _, oid := range oids {
		r, err := sd.getmulti(oid, idx)
		if err != nil {
			return out, err
		}

		for o, d := range r {
			switch {
			case strings.Contains(o, iftable+"2."):
				i := mapEntry(o, iftable+"2.")
				out[i].Descr.Value = d.OctetString
				out[i].Descr.IsSet = true
			case strings.Contains(o, ifxtable+"1."):
				i := mapEntry(o, ifxtable+"1.")
				out[i].Name.Value = d.OctetString
				out[i].Name.IsSet = true
			case strings.Contains(o, ifxtable+"18."):
				i := mapEntry(o, ifxtable+"18.")
				out[i].Alias.Value = d.OctetString
				out[i].Alias.IsSet = true
			case strings.Contains(o, iftable+"3."):
				i := mapEntry(o, iftable+"3.")
				out[i].Type.Value = d.Integer
				out[i].Type.IsSet = true
				out[i].TypeStr.Value = IfTypeStr(d.Integer)
				out[i].TypeStr.IsSet = true
			case strings.Contains(o, iftable+"4."):
				i := mapEntry(o, iftable+"4.")
				out[i].Mtu.Value = d.Integer
				out[i].Mtu.IsSet = true
			case strings.Contains(o, ifxtable+"15."):
				i := mapEntry(o, ifxtable+"15.")
				out[i].Speed.Value = d.Gauge32 * 1000000
				out[i].Speed.IsSet = true
			case strings.Contains(o, iftable+"5."):
				i := mapEntry(o, iftable+"5.")
				if out[i].Speed.IsSet {
					break
				}
				out[i].Speed.Value = d.Gauge32
				out[i].Speed.IsSet = true
			case strings.Contains(o, iftable+"6."):
				i := mapEntry(o, iftable+"6.")
				v := fmt.Sprintf("% X", d.OctetString)
				v = strings.Replace(v, " ", ":", -1)
				out[i].Mac.Value = v
				out[i].Mac.IsSet = true
			case strings.Contains(o, iftable+"7."):
				i := mapEntry(o, iftable+"7.")
				out[i].Admin.Value = d.Integer
				out[i].Admin.IsSet = true
				out[i].AdminStr.Value = IfStatStr(d.Integer)
				out[i].AdminStr.IsSet = true
			case strings.Contains(o, iftable+"8."):
				i := mapEntry(o, iftable+"8.")
				out[i].Oper.Value = d.Integer
				out[i].Oper.IsSet = true
				out[i].OperStr.Value = IfStatStr(d.Integer)
				out[i].OperStr.IsSet = true
			case strings.Contains(o, iftable+"9."):
				i := mapEntry(o, iftable+"9.")
				out[i].Last.Value = d.TimeTicks
				out[i].Last.IsSet = true
				out[i].LastStr.Value = "unkn"
				out[i].LastStr.IsSet = true
				if sut.UpTime.IsSet {
					dt := UpTimeString(sut.UpTime.Value, d.TimeTicks)
					out[i].LastStr.Value = dt
				}
			case strings.Contains(o, ifxtable+"6."):
				i := mapEntry(o, ifxtable+"6.")
				out[i].InOctets.Value = d.Counter64
				out[i].InOctets.IsSet = true
			case strings.Contains(o, iftable+"10."):
				i := mapEntry(o, iftable+"10.")
				if out[i].InOctets.IsSet {
					break
				}
				out[i].InOctets.Value = d.Counter32
				out[i].InOctets.IsSet = true
			case strings.Contains(o, ifxtable+"7."):
				i := mapEntry(o, ifxtable+"7.")
				out[i].InUcast.Value = d.Counter64
				out[i].InUcast.IsSet = true
			case strings.Contains(o, iftable+"11."):
				i := mapEntry(o, iftable+"11.")
				if out[i].InUcast.IsSet {
					break
				}
				out[i].InUcast.Value = d.Counter32
				out[i].InUcast.IsSet = true
			case strings.Contains(o, ifxtable+"8."):
				i := mapEntry(o, ifxtable+"8.")
				out[i].InMcast.Value = d.Counter64
				out[i].InMcast.IsSet = true
			case strings.Contains(o, ifxtable+"9."):
				i := mapEntry(o, ifxtable+"9.")
				out[i].InBcast.Value = d.Counter64
				out[i].InBcast.IsSet = true
			case strings.Contains(o, iftable+"13."):
				i := mapEntry(o, iftable+"13.")
				out[i].InDiscards.Value = d.Counter32
				out[i].InDiscards.IsSet = true
			case strings.Contains(o, iftable+"14."):
				i := mapEntry(o, iftable+"14.")
				out[i].InErrors.Value = d.Counter32
				out[i].InErrors.IsSet = true
			case strings.Contains(o, ifxtable+"10."):
				i := mapEntry(o, ifxtable+"10.")
				out[i].OutOctets.Value = d.Counter64
				out[i].OutOctets.IsSet = true
			case strings.Contains(o, iftable+"16."):
				i := mapEntry(o, iftable+"16.")
				if out[i].OutOctets.IsSet {
					break
				}
				out[i].OutOctets.Value = d.Counter32
				out[i].OutOctets.IsSet = true
			case strings.Contains(o, ifxtable+"11."):
				i := mapEntry(o, ifxtable+"11.")
				out[i].OutUcast.Value = d.Counter64
				out[i].OutUcast.IsSet = true
			case strings.Contains(o, iftable+"17."):
				i := mapEntry(o, iftable+"17.")
				if out[i].OutUcast.IsSet {
					break
				}
				out[i].OutUcast.Value = d.Counter32
				out[i].OutUcast.IsSet = true
			case strings.Contains(o, ifxtable+"12."):
				i := mapEntry(o, ifxtable+"12.")
				out[i].OutMcast.Value = d.Counter64
				out[i].OutMcast.IsSet = true
			case strings.Contains(o, ifxtable+"13."):
				i := mapEntry(o, ifxtable+"13.")
				out[i].OutBcast.Value = d.Counter64
				out[i].OutBcast.IsSet = true
			case strings.Contains(o, iftable+"19."):
				i := mapEntry(o, iftable+"19.")
				out[i].OutDiscards.Value = d.Counter32
				out[i].OutDiscards.IsSet = true
			case strings.Contains(o, iftable+"20."):
				i := mapEntry(o, iftable+"20.")
				out[i].OutErrors.Value = d.Counter32
				out[i].OutErrors.IsSet = true
			}
		}
	}
	return out, nil
}

// Get info from .iso.org.dod.internet.mgmt.mib-2.ifMIB.ifMIBObjects.ifStackTable
// Returns information on which sub-layers run below or on top of other sub-layers,
// where each sub-layer corresponds to a conceptual row in the ifTable.
func (sd *snmpCommon) IfStack() (IfStack, error) {
	var out IfStack
	var down = make(map[int][]int)
	var up = make(map[int][]int)

	oid := ".1.3.6.1.2.1.31.1.2.1.3"

	r, err := sd.snmpSession.Walk(oid, true, true)
	if err != nil && sd.handleErr(oid, err) {
		return out, err
	}

	for k, v := range r {
		if v.Integer != 1 {
			continue
		}

		ifIdxs := strings.Split(k, ".")
		if len(ifIdxs) != 2 || ifIdxs[0] == "0" || ifIdxs[1] == "0" {
			continue
		}

		ifIdx1, _ := strconv.Atoi(ifIdxs[0])
		ifIdx2, _ := strconv.Atoi(ifIdxs[1])

		if down[ifIdx1] == nil {
			down[ifIdx1] = []int{}
		}
		if up[ifIdx2] == nil {
			up[ifIdx2] = []int{}
		}
		down[ifIdx1] = append(down[ifIdx1], ifIdx2)
		up[ifIdx2] = append(up[ifIdx2], ifIdx1)
	}

	out.Down = down
	out.Up = up

	return out, nil
}

// Get info from .iso.org.dod.internet.mgmt.mib-2.entityMIB.entityMIBObjects.entityPhysical.entPhysicalTable
// Valid targets values: "All", "Descr", "Position", "HwProduct", "HwRev", "Serial", "Manufacturer",
//"Model", "SwProduct", "SwRev", "ParentId"
func (sd *snmpCommon) InvInfo(targets []string, idx ...string) (map[string]*InvInfo, error) {
	out := make(map[string]*InvInfo)
	invTable := ".1.3.6.1.2.1.47.1.1.1.1."
	var oids []string

	allOids := []string{
		invTable + "2", invTable + "4", invTable + "7", invTable + "8", invTable + "9", invTable + "10",
		invTable + "11", invTable + "12", invTable + "13",
	}

	for _, t := range targets {
		switch t {
		case "All":
			oids = allOids
			continue
		case "Descr":
			oids = append(oids, invTable+"2")
		case "ParentId":
			oids = append(oids, invTable+"4")
		case "Position":
			oids = append(oids, invTable+"7")
		case "HwRev":
			oids = append(oids, invTable+"8")
		case "SwProduct":
			oids = append(oids, invTable+"9")
		case "SwRev":
			oids = append(oids, invTable+"10")
		case "Serial":
			oids = append(oids, invTable+"11")
		case "Manufacturer":
			oids = append(oids, invTable+"12")
		case "HwProduct":
			oids = append(oids, invTable+"13")
		}
	}

	mapEntry := func(oid, prefix string) string {
		i := strings.TrimPrefix(oid, prefix)
		if out[i] == nil {
			out[i] = new(InvInfo)
		}
		return i
	}

	// Printable ASCII char
	reAsciiPrnt := regexp.MustCompile(`^[ -~]+$`)
	// Zero lenght or empty
	reEmpty := regexp.MustCompile(`^(\s+|empty)$`)

	for _, oid := range oids {
		r, err := sd.getmulti(oid, idx)
		if err != nil {
			return out, err
		}

		for o, d := range r {
			switch {
			case strings.Contains(o, invTable+"2."):
				i := mapEntry(o, invTable+"2.")
				if reAsciiPrnt.Match([]byte(d.OctetString)) {
					out[i].Descr.Value = d.OctetString
					out[i].Descr.IsSet = true
				}
			case strings.Contains(o, invTable+"4."):
				i := mapEntry(o, invTable+"4.")
				out[i].ParentId.Value = d.Integer
				out[i].ParentId.IsSet = true
			case strings.Contains(o, invTable+"7."):
				i := mapEntry(o, invTable+"7.")
				if reAsciiPrnt.Match([]byte(d.OctetString)) {
					out[i].Position.Value = d.OctetString
					out[i].Position.IsSet = true
				}
			case strings.Contains(o, invTable+"8."):
				i := mapEntry(o, invTable+"8.")
				if reAsciiPrnt.Match([]byte(d.OctetString)) {
					out[i].HwRev.Value = d.OctetString
					out[i].HwRev.IsSet = true
					out[i].Physical = true

				}
			case strings.Contains(o, invTable+"9."):
				i := mapEntry(o, invTable+"9.")
				if reAsciiPrnt.Match([]byte(d.OctetString)) {
					out[i].SwProduct.Value = d.OctetString
					out[i].SwProduct.IsSet = true
				}
			case strings.Contains(o, invTable+"10."):
				i := mapEntry(o, invTable+"10.")
				if reAsciiPrnt.Match([]byte(d.OctetString)) {
					out[i].SwRev.Value = d.OctetString
					out[i].SwRev.IsSet = true
				}
			case strings.Contains(o, invTable+"11."):
				i := mapEntry(o, invTable+"11.")
				if reAsciiPrnt.Match([]byte(d.OctetString)) {
					out[i].Serial.Value = d.OctetString
					out[i].Serial.IsSet = true
					out[i].Physical = true
				}
			case strings.Contains(o, invTable+"12."):
				i := mapEntry(o, invTable+"12.")
				if reAsciiPrnt.Match([]byte(d.OctetString)) {
					out[i].Manufacturer.Value = d.OctetString
					out[i].Manufacturer.IsSet = true
				}
			case strings.Contains(o, invTable+"13."):
				i := mapEntry(o, invTable+"13.")
				if reAsciiPrnt.Match([]byte(d.OctetString)) {
					out[i].HwProduct.Value = d.OctetString
					out[i].HwProduct.IsSet = true
					out[i].Model.Value = d.OctetString
					out[i].Model.IsSet = true
					if !reEmpty.Match([]byte(d.OctetString)) {
						out[i].Physical = true
					}
				}
			}
		}
	}

	return out, nil
}

// Get info from .iso.org.dod.internet.mgmt.mib-2.entityMIB.entityMIBObjects.entityMapping.entAliasMappingTable
// Returns ifIndex to entityId relations map.
func (sd *snmpCommon) IfInventory() (map[int]int, error) {
	var out = make(map[int]int)

	oid := ".1.3.6.1.2.1.47.1.3.2.1.2"

	r, err := sd.snmpSession.Walk(oid, true, true)
	if err != nil && sd.handleErr(oid, err) {
		return out, err
	}

	for k, v := range r {
		ePart := strings.Split(k, ".")
		ei := ePart[0]
		iPart := strings.Split(v.ObjectIdentifier, ".")
		ii := iPart[len(iPart)-1]

		eIdx, _ := strconv.Atoi(ei)
		iIdx, _ := strconv.Atoi(ii)

		out[iIdx] = eIdx
	}

	return out, nil
}

// Get info from .iso.org.dod.internet.mgmt.mib-2.dot1dBridge.dot1dBase.dot1dBasePortTable
// Returns bridgeport index to ifindex map
func (sd *snmpCommon) BrPort2IfIdx() (map[string]int, error) {
	var out = make(map[string]int)

	oid := ".1.3.6.1.2.1.17.1.4.1.2"

	r, err := sd.snmpSession.Walk(oid, true, true)
	if err != nil && sd.handleErr(oid, err) {
		return out, err
	}

	for k, v := range r {
		out[k] = int(v.Integer)
	}

	return out, nil
}

// Get info from .iso.org.dod.internet.mgmt.mib-2.dot1dBridge.qBridgeMIB.qBridgeMIBObjects.dot1qVlan.dot1qVlanStaticTable
// Returns vlan id-s and names
func (sd *snmpCommon) D1qVlans() (map[string]string, error) {
	var out = make(map[string]string)

	oid := ".1.3.6.1.2.1.17.7.1.4.3.1.1"

	r, err := sd.snmpSession.Walk(oid, true, true)
	if err != nil && sd.handleErr(oid, err) {
		return out, err
	}

	for k, v := range r {
		out[k] = v.OctetString
	}

	return out, nil
}

// Get info from .iso.org.dod.internet.mgmt.mib-2.dot1dBridge.qBridgeMIB.qBridgeMIBObjects.dot1qVlan.dot1qVlanCurrentTable
func (sd *snmpCommon) D1qVlanInfo() (map[string]*D1qVlanInfo, error) {
	out := make(map[string]*D1qVlanInfo)
	mOids := []string{".1.3.6.1.2.1.17.7.1.4.3.1.2", ".1.3.6.1.2.1.17.7.1.4.2.1.4"}
	uOids := []string{".1.3.6.1.2.1.17.7.1.4.3.1.4", ".1.3.6.1.2.1.17.7.1.4.2.1.5"}

	// get vlans
	vlans, err := sd.D1qVlans()
	if err != nil {
		return out, err
	}

	// get vlans
	ifIdx, err := sd.BrPort2IfIdx()
	if err != nil {
		return out, err
	}

	// get vlan member ports
	for i, mOid := range mOids {
		mr, err := sd.snmpSession.Walk(mOid, true, true)
		if err != nil {
			if i == 0 {
				continue
			}
			if sd.handleErr(mOid, err) {
				return out, fmt.Errorf("%s - %s", mOid, err)
			}
		}

		mBytes := make(map[string][]byte)
		for k, v := range mr {
			vPart := strings.Split(k, ".")
			if len(vPart) == 2 {
				mBytes[vPart[1]] = []byte(v.OctetString)
			} else {
				mBytes[vPart[0]] = []byte(v.OctetString)
			}
		}

		for v, bytes := range mBytes {
			bMap := BitMap(bytes)

			ports := make(map[int]*D1qVlanBrPort)
			for p := range bMap {
				ports[p] = &D1qVlanBrPort{IfIdx: ifIdx[strconv.Itoa(p)]}
			}

			out[v] = &D1qVlanInfo{
				Name:  vlans[v],
				Ports: ports,
			}
		}
		break
	}

	// get vlan untagged member ports
	for i, uOid := range uOids {
		ur, err := sd.snmpSession.Walk(uOid, true, true)
		if err != nil {
			if i == 0 {
				continue
			} else if sd.handleErr(uOid, err) {
				return out, err
			}
		}

		uBytes := make(map[string][]byte)
		for k, v := range ur {
			vPart := strings.Split(k, ".")
			if len(vPart) == 2 {
				uBytes[vPart[1]] = []byte(v.OctetString)
			} else {
				uBytes[vPart[0]] = []byte(v.OctetString)
			}
		}

		for v, bytes := range uBytes {
			bMap := BitMap(bytes)

			for p := range bMap {
				out[v].Ports[p].UnTag = true
			}
		}
		break
	}

	return out, nil
}

// Get info from .iso.org.dod.internet.mgmt.mib-2.ip.ipAddrTable
func (sd *snmpCommon) IpInfo(ip ...string) (map[string]*IpInfo, error) {
	out := make(map[string]*IpInfo)
	ipTable := ".1.3.6.1.2.1.4.20.1."
	oids := []string{ipTable + "2", ipTable + "3"}

	mapEntry := func(oid, prefix string) string {
		i := strings.TrimPrefix(oid, prefix)
		i = strings.TrimPrefix(i, "0.")
		if out[i] == nil {
			out[i] = new(IpInfo)
		}
		return i
	}

	for _, oid := range oids {
		r, err := sd.getmulti(oid, ip)
		if err != nil {
			return out, err
		}

		// Some devices prepend (0.)+ to the ip
		if r == nil && ip != nil {
			r, err = sd.getmulti(oid+".0", ip)
			if err != nil {
				return out, err
			}
		}

		for o, d := range r {
			switch {
			case strings.Contains(o, ipTable+"2."):
				i := mapEntry(o, ipTable+"2.")
				out[i].IfIdx = d.Integer
			case strings.Contains(o, ipTable+"3."):
				i := mapEntry(o, ipTable+"3.")
				out[i].Mask = d.IPAddress
			}
		}
	}

	return out, nil
}

// Get IP Interface info
func (sd *snmpCommon) IpIfInfo(ip ...string) (map[string]*IpIfInfo, error) {
	out := make(map[string]*IpIfInfo)

	ipInfo, err := sd.IpInfo(ip...)
	if err != nil {
		return out, err
	}

	// Get slice of ifIndexes from ipInfo and fill output map with ip info
	ifIdxs := make([]string, 0, len(ipInfo))
	for i, v := range ipInfo {
		ifIdxs = append(ifIdxs, strconv.FormatInt(int64(v.IfIdx), 10))

		if out[i] == nil {
			out[i] = new(IpIfInfo)
		}
		out[i].IpInfo = *v
	}

	ifInfo, err := sd.IfInfo([]string{"Descr", "Alias"}, ifIdxs...)
	if err != nil {
		return out, err
	}

	// Fill output map with interface info
	for i, d := range ipInfo {
		ifIdxStr := strconv.FormatInt(int64(d.IfIdx), 10)
		out[i].Descr = ifInfo[ifIdxStr].Descr.Value
		out[i].Alias = ifInfo[ifIdxStr].Alias.Value
	}

	return out, err
}

// Get IPv6 Interface description from .iso.org.dod.internet.mgmt.mib-2.ipv6MIB.ipv6MIBObjects.ipv6IfTable
func (sd *snmpCommon) Ip6IfDescr(idx ...string) (map[string]string, error) {
	out := make(map[string]string)
	oid := ".1.3.6.1.2.1.55.1.5.1.2"

	r, err := sd.getmulti(oid, idx)
	if err != nil {
		return out, err
	}

	for o, d := range r {
		i := strings.TrimPrefix(o, oid+".")
		out[i] = d.OctetString
	}

	return out, nil
}

// Get info from .iso.org.dod.internet.mgmt.mib-2.ospf.ospfLsdbTable
// Returns OSPF area to area router relations map.
func (sd *snmpCommon) OspfAreaRouters() (map[string][]string, error) {
	var out = make(map[string][]string)

	oid := ".1.3.6.1.2.1.14.4.1.1"

	r, err := sd.snmpSession.Walk(oid, true, true)
	if err != nil && sd.handleErr(oid, err) {
		return out, err
	}

	reLastIp := regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+\.1\..*\.(\d+\.\d+\.\d+\.\d+)$`)

	for k, v := range r {
		rPart := reLastIp.FindStringSubmatch(k)
		if len(rPart) != 2 {
			continue
		}
		out[v.IPAddress] = append(out[v.IPAddress], rPart[1])
	}

	return out, nil
}

// Get info from .iso.org.dod.internet.mgmt.mib-2.ospf.ospfAreaTable
// Returns OSPF area status map.
func (sd *snmpCommon) OspfAreaStatus() (map[string]string, error) {
	var out = make(map[string]string)

	oid := ".1.3.6.1.2.1.14.2.1.10"
	r, err := sd.snmpSession.Walk(oid, true, true)
	if err != nil && sd.handleErr(oid, err) {
		return out, err
	}

	states := map[int64]string{
		1: "active",
		2: "notInService",
		3: "notReady",
		4: "createAndGo",
		5: "createAndWait",
		6: "destroy",
	}

	for k, v := range r {
		state, ok := states[v.Integer]
		if !ok {
			state = "unkn"
		}

		out[k] = state
	}

	return out, nil
}

// Get info from .iso.org.dod.internet.mgmt.mib-2.ospf.ospfAreaTable
// Returns OSPF neighbour status map.
func (sd *snmpCommon) OspfNbrStatus() (map[string]string, error) {
	var out = make(map[string]string)

	oid := ".1.3.6.1.2.1.14.10.1.6"
	r, err := sd.snmpSession.Walk(oid, true, true)
	if err != nil && sd.handleErr(oid, err) {
		return out, err
	}

	states := map[int64]string{
		1: "down",
		2: "attempt",
		3: "init",
		4: "twoWay",
		5: "exchangeStart",
		6: "exchange",
		7: "loading",
		8: "full",
	}

	reNbrIp := regexp.MustCompile(`(.*)\.\d+$`)
	for k, v := range r {
		nPart := reNbrIp.FindStringSubmatch(k)
		if len(nPart) != 2 {
			continue
		}

		state, ok := states[v.Integer]
		if !ok {
			state = "unkn"
		}

		out[nPart[1]] = state
	}

	return out, nil
}

// Set Interface Admin status
// set - map of ifIndexes and their states (up|down)
func (sd *snmpCommon) SetIfAdmStat(set map[string]string) error {
	pdus := []snmphelper.SetPDU{}
	states := map[string]int{
		"up":   1,
		"down": 2,
	}

	for i, state := range set {
		s, ok := states[state]
		if !ok {
			return fmt.Errorf("interface state %s is not valid", state)
		}

		pdu := snmphelper.SetPDU{
			Oid:   ".1.3.6.1.2.1.2.2.1.7." + i,
			Vtype: "Integer",
			Value: s,
		}
		pdus = append(pdus, pdu)
	}

	r, err := sd.snmpSession.Set(pdus)
	if err != nil {
		return err
	}

	// DEBUG
	if sd.debug > 0 {
		fmt.Printf("SetIfAdmStat result: %# v\n", pretty.Formatter(r))
	}

	return nil
}

// Set Interface Alias
// set - map of ifIndexes and related ifAliases
func (sd *snmpCommon) SetIfAlias(set map[string]string) error {
	pdus := []snmphelper.SetPDU{}

	for i, a := range set {
		pdu := snmphelper.SetPDU{
			Oid:   ".1.3.6.1.2.1.31.1.1.1.18." + i,
			Vtype: "OctetString",
			Value: a,
		}
		pdus = append(pdus, pdu)
	}

	r, err := sd.snmpSession.Set(pdus)
	if err != nil {
		return err
	}

	// DEBUG
	if sd.debug > 0 {
		fmt.Printf("SetIfAlias result: %# v\n", pretty.Formatter(r))
	}

	return nil
}

// Set Device sysName
func (sd *snmpCommon) SetSysName(v string) error {
	pdus := []snmphelper.SetPDU{
		{
			Oid:   ".1.3.6.1.2.1.1.5.0",
			Vtype: "OctetString",
			Value: v,
		},
	}

	r, err := sd.snmpSession.Set(pdus)
	if err != nil {
		return err
	}

	// DEBUG
	if sd.debug > 0 {
		fmt.Printf("SetSysName result: %# v\n", pretty.Formatter(r))
	}

	return nil
}

// Set Device contact
func (sd *snmpCommon) SetContact(v string) error {
	pdus := []snmphelper.SetPDU{
		{
			Oid:   ".1.3.6.1.2.1.1.4.0",
			Vtype: "OctetString",
			Value: v,
		},
	}

	r, err := sd.snmpSession.Set(pdus)
	if err != nil {
		return err
	}

	// DEBUG
	if sd.debug > 0 {
		fmt.Printf("SetContact result: %# v\n", pretty.Formatter(r))
	}

	return nil
}

// Set Device location
func (sd *snmpCommon) SetLocation(v string) error {
	pdus := []snmphelper.SetPDU{
		{
			Oid:   ".1.3.6.1.2.1.1.6.0",
			Vtype: "OctetString",
			Value: v,
		},
	}

	r, err := sd.snmpSession.Set(pdus)
	if err != nil {
		return err
	}

	// DEBUG
	if sd.debug > 0 {
		fmt.Printf("SetLocation result: %# v\n", pretty.Formatter(r))
	}

	return nil
}
