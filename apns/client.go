package apns

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/http2"
)

const (
	// HTTP2 client timeout
	HTTP2ClientTimeout = time.Second * 10
)

// APNsClient is Client
type Client struct {
	Host   string
	Client *http.Client
}

// NewAPNsClient returns APNsClient
func NewAPNsClient(host string, c *http.Client) Client {
	return Client{
		Host:   host,
		Client: c,
	}
}

// Send sends notifications to apns
func (ac *Client) Send(req Request) (*Response, error) {
	areq := req.Request().(Request)
	nreq, err := ac.NewRequest(areq.Token, &areq.Header, areq.Payload)
	if err != nil {
		return nil, err
	}

	res, err := ac.Client.Do(nreq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	ret := &Response{
		APNsID:     res.Header.Get("apns-id"),
		StatusCode: res.StatusCode,
	}

	if res.StatusCode != http.StatusOK {
		var er ErrorResponse
		body, err := ioutil.ReadAll(res.Body)
		// ioutil Error
		if err != nil {
			return ret, err
		}
		// Unmarshal Error
		if err := json.Unmarshal(body, &er); err != nil {
			return ret, err
		}
		return ret, &er
	}

	return ret, nil
}

// NewRequest creates request for apns
func (ac *Client) NewRequest(token string, h *Header, payload Payload) (*http.Request, error) {
	u, err := url.Parse(fmt.Sprintf("%s/3/device/%s", ac.Host, token))
	if err != nil {
		return nil, err
	}

	data, err := payload.MarshalJSON()
	if err != nil {
		return nil, err
	}

	nreq, err := http.NewRequest("POST", u.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	if h != nil {
		if h.ApnsID != "" {
			nreq.Header.Set("apns-id", h.ApnsID)
		}
		if h.ApnsExpiration != "" {
			nreq.Header.Set("apns-expiration", h.ApnsExpiration)
		}
		if h.ApnsPriority != "" {
			nreq.Header.Set("apns-priority", h.ApnsPriority)
		}
		if h.ApnsTopic != "" {
			nreq.Header.Set("apns-topic", h.ApnsTopic)
		}
	}

	return nreq, err
}

// NewConnection establishes a http2 connection
func NewConnection(certFile, keyFile string, secuskip bool) (*http.Client, error) {
	certPEMBlock, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	keyPEMBlock, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)

	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: secuskip,
			Certificates:       []tls.Certificate{cert},
		},
	}

	if err := http2.ConfigureTransport(tr); err != nil {
		return nil, err
	}

	return &http.Client{
		Timeout:   HTTP2ClientTimeout,
		Transport: tr,
	}, nil
}
