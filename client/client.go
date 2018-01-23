package client

import (
	"bytes"
	"encoding/base64"
	"fmt"
	//"github.com/miekg/dns"
	"log"
	"net"
	"net/http"
	//"github.com/patrickmn/go-cache"
)

type client struct {
	cfg        Config
	listenIP   net.IP
	listenPort int
	httpClient *http.Client
}

const (
	// TypeDNSUDPWireFormat ...
	TypeDNSUDPWireFormat = "application/dns-udpwireformat"
)

// Client ...
type Client interface {
	ListenAndServe() error
}

// NewClient makes a new client
func NewClient(cfg Config) (Client, error) {
	laddr := net.ParseIP(cfg.ListenAddr)
	if laddr == nil {
		return nil, fmt.Errorf("couldnt parse listenaddress")
	}
	lport := cfg.ListenPort
	httpClient := http.Client{}
	c := client{
		cfg:        cfg,
		listenIP:   laddr,
		listenPort: lport,
		httpClient: &httpClient,
	}
	return &c, nil
}

// ListenAndServe handles DNS over HTTP requests
func (c *client) ListenAndServe() error {
	listenAddr := net.UDPAddr{
		IP:   c.listenIP,
		Port: c.listenPort,
	}
	log.Printf("Listening for DNS on %s:%d\n", c.listenIP.String(), c.listenPort)
	udpconn, err := net.ListenUDP("udp", &listenAddr)
	if err != nil {
		return err
	}

	var buf []byte
	for {
		buf = make([]byte, 1500)
		nread, rAddr, err := udpconn.ReadFromUDP(buf)
		if err == nil {
			// TODO: thread this
			err = c.handle(buf[0:nread], rAddr, udpconn)
			if err != nil {
				log.Printf("error handling request: %v\n", err)
			}
		}
	}

}

// handle handles a UDP datagram request
func (c *client) handle(buf []byte, ua *net.UDPAddr, conn *net.UDPConn) error {

	//lets just blast the wire request into the HTTP request

	// body=base64url(wireformat) IFF method=GET
	base64dnsreq := []byte{} // i dunno how large to make this? can we presize this? i suspect so but for now, hackday.
	base64.URLEncoding.Encode(base64dnsreq, buf)

	// path: /.well-known/dns-query
	//NOTE: using GET becuase its cachable
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/.well-known/dns-query", c.cfg.DOHURL),
		nil)
	if err != nil {
		return err
	}

	// set params for the request
	params := req.URL.Query()
	params.Add("ct", TypeDNSUDPWireFormat)
	params.Add("body", bytes.NewBuffer(base64dnsreq).String())
	req.URL.RawQuery = params.Encode()

	// set headers, etc
	req.Header.Set("accept", TypeDNSUDPWireFormat)
	req.Header.Set("content-type", TypeDNSUDPWireFormat)

	/*
	   :method = POST
	   :scheme = https
	   :authority = dnsserver.example.net
	   :path = /.well-known/dns-query
	   accept = application/dns-udpwireformat, application/simpledns+json
	   content-type = application/dns-udpwireformat
	   content-length = 33

	   <33 bytes represented by the following hex encoding>
	   abcd 0100 0001 0000 0000 0000 0377 7777
	   0765 7861 6d70 6c65 0363 6f6d 0000 0100
	   01
	*/

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	// process resp
	length := resp.Header.Get("content-length")
	log.Print("got response with content-length: %s\n", length)
	if resp.Header.Get("content-type") != TypeDNSUDPWireFormat {
		return fmt.Errorf("response had wrong content type! unable to process %v", resp.Header.Get("content-type"))
	}
	bufResp := []byte{}
	nBytes, err := resp.Body.Read(bufResp)
	if err != nil {
		return err
	}
	log.Printf("read %d bytes from body\n", nBytes)
	log.Printf("body: %v\n", bufResp)

	nBytes, err = conn.WriteToUDP(bufResp, ua)
	if err != nil {
		return err
	}
	log.Printf("wrote %d bytes of response back to %s\n", nBytes, ua)
	return nil
}
