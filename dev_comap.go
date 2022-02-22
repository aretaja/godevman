package godevman

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
