package godevman

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

// Adds ECS energy-meter device functionality to device type
type deviceEcsEmeter struct {
	device
}

// ECS specific system info type used by device web API
type EcsStatus1 struct {
	XMLName  xml.Name `xml:"root"`
	Chardata string   `xml:",chardata"`
	Name     string   `xml:"name"`
	Type     string   `xml:"type"`
	Class    string   `xml:"class"`
	Text     string   `xml:"text"`
}

//
type EcsReadings1 struct {
	XMLName xml.Name `xml:"root"`
	Text    string   `xml:",chardata"`
	Par     string   `xml:"par"`
	Data    []string `xml:"d"`
}

// Make http GET request and return byte slice of body.
// Argument string should contain request parameters.
func (sd *deviceEcsEmeter) WebApiGet(params string) ([]byte, error) {
	req := "GET /" + params + " HTTP/1.1\r\n" +
		"Host: " + sd.ip + "\r\n" +
		"User-Agent: godevman\r\n\r\n"

	res, err := TcpReq(req, sd.ip, "80")
	if err != nil {
		return nil, err
	}

	reParts := regexp.MustCompile(`(?s)^(.*?)\r\n\r\n(.*)`)
	parts := reParts.FindStringSubmatch(string(res))

	if parts != nil && strings.HasPrefix(parts[0], "HTTP/") && len(parts) > 2 {
		return []byte(strings.Join(parts[2:], "")), nil
	}

	return res, nil
}

// Identify ECS verion (v1 or v2)
func (sd *deviceEcsEmeter) ecsVersion() (string, error) {
	// return from cache if allowed and cache is present
	if sd.useCache {
		if x, found := sd.cache.Get("ecsVersion"); found {
			return x.(string), nil
		}
	}

	var out string
	res, err := sd.WebApiGet("status.xml")
	if err != nil {
		return out, fmt.Errorf("get request from device api failed: %s", err)
	}

	if strings.Contains(string(res), "Energy Meter") {
		out = "v1"
	}

	if out != "v1" {
		res, err := sd.WebApiGet("login.cgi")
		if err != nil {
			return out, fmt.Errorf("get request from device api failed: %s", err)
		}

		if strings.Contains(string(res), "granted") {
			out = "v2"
		} else {
			return out, fmt.Errorf("ecs identify failed")
		}
	}

	// save to cache
	sd.cache.Set("ecsVersion", out, cache.DefaultExpiration)

	return out, nil
}

// Get info from web
// Valid targets values: "All", "Descr", "ObjectID", "Name"
func (sd *deviceEcsEmeter) System(targets []string) (System, error) {
	out := System{
		ObjectID: ValString{
			IsSet: true,
			Value: "no-snmp-ecs",
		},
	}

	ver, err := sd.ecsVersion()
	if err != nil {
		return out, fmt.Errorf("ecsVersionerror: %s", err)
	}

	if ver == "v2" {
		out.Descr.Value = "Energy Meter v2"
		out.Descr.IsSet = true
		return out, nil
	}

	if ver == "v1" {
		res, err := sd.WebApiGet("status.xml")
		if err != nil {
			return out, fmt.Errorf("ecsVersion error: %s", err)
		}

		var v EcsStatus1
		err = xml.Unmarshal([]byte(res), &v)
		if err != nil {
			return out, fmt.Errorf("xml unmarshal error: %s", err)
		}

		for _, t := range targets {
			if t == "All" || t == "Descr" {
				out.Descr.Value = v.Type
				out.Descr.IsSet = true
			}
			if t == "All" || t == "Name" {
				out.Name.Value = v.Name
				out.Name.IsSet = true
			}
		}
	}

	return out, err
}

// Get energy redings
func (sd *deviceEcsEmeter) Ereadings() (*EReadings, error) {
	ver, err := sd.ecsVersion()
	if err != nil {
		return nil, fmt.Errorf("ecsVersion error: %s", err)
	}

	out := EReadings{
		timeStamp: uint(time.Now().Unix()),
	}

	if ver == "v1" {
		res, err := sd.WebApiGet("values.xml")
		if err != nil {
			return nil, fmt.Errorf("get readings error: %s", err)
		}

		var v EcsReadings1
		err = xml.Unmarshal([]byte(res), &v)
		if err != nil {
			return nil, fmt.Errorf("xml unmarshal error: %s", err)
		}

		d := strings.Replace(v.Data[4], ",", ".", 1)
		n := strings.Replace(v.Data[5], ",", ".", 1)

		df, err := strconv.ParseFloat(d, 64)
		if err != nil {
			return nil, fmt.Errorf("ParseFloat day value error: %s", err)
		}

		nf, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return nil, fmt.Errorf("ParseFloat night value error: %s", err)
		}

		out.day.Value = uint64(df * 10000)
		out.day.Divisor = 10000
		out.day.Unit = "kWh"
		out.day.IsSet = true

		out.night.Value = uint64(nf * 10000)
		out.night.Divisor = 10000
		out.night.Unit = "kWh"
		out.night.IsSet = true

		return &out, nil
	}

	if ver == "v2" {
		res, err := sd.WebApiGet("eVision.cgi?dataType=HOME&t=" + RandomString(8))
		if err != nil {
			return nil, fmt.Errorf("get readings error: %s", err)
		}

		reDay := regexp.MustCompile(`(?s)ActEnergyT1imp=([\d\.]+)`)
		dParts := reDay.FindStringSubmatch(string(res))
		reNight := regexp.MustCompile(`(?s)ActEnergyT2imp=([\d\.]+)`)
		nParts := reNight.FindStringSubmatch(string(res))

		if len(dParts) == 2 {
			df, err := strconv.ParseFloat(dParts[1], 64)
			if err != nil {
				return nil, fmt.Errorf("ParseFloat day value error: %s", err)
			}

			out.day.Value = uint64(df * 10000)
			out.day.Divisor = 10000
			out.day.Unit = "kWh"
			out.day.IsSet = true
		}

		if len(nParts) == 2 {
			nf, err := strconv.ParseFloat(nParts[1], 64)
			if err != nil {
				return nil, fmt.Errorf("ParseFloat night value error: %s", err)
			}

			out.night.Value = uint64(nf * 10000)
			out.night.Divisor = 10000
			out.night.Unit = "kWh"
			out.night.IsSet = true
		}

		return &out, nil
	}

	return nil, fmt.Errorf("readings not found")
}
