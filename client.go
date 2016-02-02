package soap

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

// ClientDialTimeout default timeout 30s
var ClientDialTimeout = time.Duration(30 * time.Second)

// UserAgent is the default user agent
var UserAgent = "go-soap-0.1"

// Verbose be verbose
var Verbose = false

func l(m ...interface{}) {
	if Verbose {
		log.Println(m...)
	}
}

// XMLMarshaller lets you inject your favourite custom xml implementation
type XMLMarshaller interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(xml []byte, v interface{}) error
}

type defaultMarshaller struct {
}

func (dm *defaultMarshaller) Marshal(v interface{}) (xmlBytes []byte, err error) {
	return xml.Marshal(v)
}

func (dm *defaultMarshaller) Unmarshal(xmlBytes []byte, v interface{}) error {
	return xml.Unmarshal(xmlBytes, v)
}

func newDefaultMarshaller() XMLMarshaller {
	return &defaultMarshaller{}
}

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, ClientDialTimeout)
}

// BasicAuth credentials for the client
type BasicAuth struct {
	Login    string
	Password string
}

// Client generic SOAP client
type Client struct {
	url        string
	tls        bool
	auth       *BasicAuth
	tr         *http.Transport
	Marshaller XMLMarshaller
}

// NewClient constructor
func NewClient(url string, auth *BasicAuth, tr *http.Transport) *Client {
	return &Client{
		url:        url,
		auth:       auth,
		tr:         tr,
		Marshaller: newDefaultMarshaller(),
	}
}

// Call make a SOAP call
func (s *Client) Call(soapAction string, request, response interface{}) (httpResponse *http.Response, err error) {
	envelope := Envelope{}

	envelope.Body.Content = request

	xmlBytes, err := s.Marshaller.Marshal(envelope)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", s.url, bytes.NewBuffer(xmlBytes))
	if err != nil {
		return
	}
	if s.auth != nil {
		req.SetBasicAuth(s.auth.Login, s.auth.Password)
	}

	req.Header.Add("Content-Type", "text/xml; charset=\"utf-8\"")
	req.Header.Set("User-Agent", UserAgent)

	if soapAction != "" {
		req.Header.Add("SOAPAction", soapAction)
	}

	req.Close = true
	tr := s.tr
	if tr == nil {
		tr = http.DefaultTransport.(*http.Transport)
	}
	client := &http.Client{Transport: tr}
	l("POST to", s.url, "with", string(xmlBytes))
	httpResponse, err = client.Do(req)
	if err != nil {
		return
	}
	defer httpResponse.Body.Close()

	rawbody, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return
	}
	if len(rawbody) == 0 {
		l("empty response")
		return
	}

	l("response", string(rawbody))
	respEnvelope := new(Envelope)
	respEnvelope.Body = Body{Content: response}
	err = xml.Unmarshal(rawbody, respEnvelope)
	if err != nil {
		return
	}

	fault := respEnvelope.Body.Fault
	if fault != nil {
		err = fault
		return
	}
	return
}