package client

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
	//"github.com/patrickmn/go-cache"

	"github.com/byxorna/homer/types"
)

type client struct {
	cfg        Config
	listenIP   net.IP
	listenPort int
	httpClient *http.Client
}

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
			tStart := time.Now()
			err = c.handle(buf[0:nread], rAddr, udpconn)
			tEnd := time.Now()
			elapsed := tEnd.Sub(tStart)
			if err != nil {
				log.Printf("error handling request: %v\n", err)
			}
			log.Printf("Handled request in %s\n", elapsed.String())
		}
	}

}

// handle handles a UDP datagram request
func (c *client) handle(buf []byte, ua *net.UDPAddr, conn *net.UDPConn) error {

	//lets just blast the wire request into the HTTP request

	// body=base64url(wireformat) IFF method=GET
	// each 6 bits -> 1 byte
	base64dnsreq := make([]byte, base64.URLEncoding.EncodedLen(len(buf)))
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
	params.Add("ct", types.TypeDNSUDPWireFormat)
	params.Add("body", bytes.NewBuffer(base64dnsreq).String())
	req.URL.RawQuery = params.Encode()

	// set headers, etc
	req.Header.Set("accept", types.TypeDNSUDPWireFormat)
	req.Header.Set("content-type", types.TypeDNSUDPWireFormat)

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
	log.Printf("got %s response with content-length: %s\n", resp.Status, length)
	if resp.Header.Get("content-type") != types.TypeDNSUDPWireFormat {
		return fmt.Errorf("response had wrong content type! unable to process %v", resp.Header.Get("content-type"))
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("bummer, got %s\n", resp.Status)
		// TODO: should return a simulated message to resolver to indicate SERVFAIL
		return fmt.Errorf("fuck something was wrong with the request")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Printf("read %d bytes from body\n", len(body))
	//log.Printf("body: %v\n", body)

	nBytes, err := conn.WriteToUDP(body, ua)
	if err != nil {
		return err
	}
	log.Printf("wrote %d bytes of response back to %s\n", nBytes, ua)
	return nil
}
