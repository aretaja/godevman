package godevman

import (
	"log"
	"strings"

	"github.com/aretaja/snmphelper"
)

// Single oid get helper
func (sd *snmpCommon) getone(oid string) (snmphelper.SnmpOut, error) {
	o := []string{oid}
	res, err := sd.snmpsession.Get(o)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Walk or multiindex get helper
func (sd *snmpCommon) getmulti(oid string, idx []string) (snmphelper.SnmpOut, error) {
	if idx != nil {
		res := make(snmphelper.SnmpOut)

		var alloids []string
		for _, i := range idx {
			alloids = append(alloids, oid+"."+i)
		}

		// Don't query more oids than "MaxRepetitions" value or 5 if 0
		maxOids := int(sd.snmpsession.MaxRepetitions)
		if maxOids < 1 {
			maxOids = 5
		}

		var t int
		l := len(alloids)
		for f := 0; f < l; f += maxOids {
			t += maxOids
			if t >= l {
				t = l
			}

			oids := alloids[f:t]
			r, err := sd.snmpsession.Get(oids)
			if err != nil {
				return nil, err
			}

			for k, v := range r {
				res[k] = v
			}
		}

		return res, nil
	}

	res, err := sd.snmpsession.Walk(oid, true, false)
	if err != nil && sd.handleErr(oid, err) {
		return nil, err
	}

	return res, nil
}

// Handle snmpwalk errors
func (sd *snmpCommon) handleErr(oid string, err error) bool {
	errStr := err.Error()

	if strings.HasSuffix(errStr, "NoSuchName") ||
		strings.HasSuffix(errStr, "no results") {
		log.Printf("warning: %s\n", errStr)
		return false
	}
	return true
}
