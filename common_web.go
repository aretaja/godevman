package godevman

import (
	"crypto/tls"
	"fmt"
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
	return d.websession.client
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
		Timeout:   time.Second * 10,
		Transport: MyRoundTripper{r: tr, h: headers},
		Jar:       jar,
	}

	return client, nil
}
