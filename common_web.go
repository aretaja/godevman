package godevman

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// Add additional default headers
type MyRoundTripper struct {
	r http.RoundTripper
	h map[string][]string
}

func (rt MyRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	for k, v := range rt.h {
		r.Header[k] = v
	}

	return rt.r.RoundTrip(r)
}

// Get authenticated web session
func (d *device) WebSession() *http.Client {
	return d.webSession.client
}

// Create http client
func (d *device) webClient(headers map[string][]string) (*http.Client, error) {
	// disable certificate check
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// setup cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("got error while creating cookie jar %s", err.Error())
	}

	// return client
	client := &http.Client{
		Timeout:   time.Second * 15,
		Transport: MyRoundTripper{r: tr, h: headers},
		Jar:       jar,
	}

	return client, nil
}

// Make http Get request and return byte slice of body.
// Argument string should contain request parameters.
func (d *device) WebApiGet(params string) ([]byte, error) {
	client := d.webSession.client
	if d.webSession.client == nil {
		// setup client
		c, err := d.webClient(nil)
		if err != nil {
			return nil, err
		}
		client = c
	}

	res, err := client.Get("https://" + d.ip + "/" + params)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode > 299 {
		return body, fmt.Errorf("response failed with status code: %d", res.StatusCode)
	}

	return body, nil
}
