package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/byxorna/homer/pkg/types"
	"github.com/miekg/dns"
	"github.com/patrickmn/go-cache"
)

type client struct {
	cfg        Config
	listenIP   net.IP
	listenPort int
	httpClient *http.Client
	cache      *cache.Cache
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
	if cfg.DOH.CAFile != "" {
		// if provided, use a custom CA
		caCert, err := ioutil.ReadFile(cfg.DOH.CAFile)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		// Setup HTTPS client
		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}
		tlsConfig.BuildNameToCertificate()
		httpClient = http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
	}

	c := client{
		cfg:        cfg,
		listenIP:   laddr,
		listenPort: lport,
		httpClient: &httpClient,
		cache:      cache.New(24*time.Hour, 60*time.Second),
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

	// validate this is an ok request first
	reqMsg := dns.Msg{}
	err := reqMsg.Unpack(buf)
	if err != nil {
		log.Printf("not valid DNS request: %v\n", err)
		return err
	}
	if len(reqMsg.Question) < 1 {
		return fmt.Errorf("no questions in request")
	}

	qname := reqMsg.Question[0].Name
	qtype := dns.TypeToString[reqMsg.Question[0].Qtype]
	qid := fmt.Sprintf("%s|%s", qtype, qname)
	log.Printf("resolving %s", qid)

	// handle returning results from cache if present
	// lifted from https://github.com/pforemski/dingo/blob/master/dingo.go#L100
	if x, found := c.cache.Get(qid); found {
		// FIXME: update TTLs
		log.Printf("cached entry for %s found\n", qid)
		cachedData := x.([]byte)
		//unpack message and update the header ID to match the request
		var cachedMsg dns.Msg
		err := cachedMsg.Unpack(cachedData)
		if err == nil {
			cachedMsg.Id = reqMsg.Id
			// now, repack the message and transmit to the udpaddr
			repacked, err := cachedMsg.Pack()
			if err == nil {
				_, err := conn.WriteToUDP(repacked, ua)
				if err != nil {
					log.Printf("error writing cached response for %s to %s: %v\n", qid, ua, err)
				}
				return err
			}
			log.Printf("error repacking cached message %s with id %d; invalidating cache and continuing: %v\n", qid, reqMsg.Id, err)
			c.cache.Delete(qid)
		} else {
			log.Printf("error unpacking cached message %s; invalidating cache and continuing: %v\n", qid, err)
			c.cache.Delete(qid)
		}
	}

	// body=base64url(wireformat) IFF method=GET
	base64dnsreq := make([]byte, base64.URLEncoding.EncodedLen(len(buf)))
	base64.URLEncoding.Encode(base64dnsreq, buf)

	// path: /.well-known/dns-query
	//NOTE: using GET becuase its cachable
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/.well-known/dns-query", c.cfg.DOH.URL),
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
	//log.Printf("read %d bytes from body\n", len(body))

	/* put resp in cache for 10 seconds (FIXME: use minimum TTL) */
	// TODO: make the cached value = msg.Answer[0].Ttl
	c.cache.Set(qid, body, 10*time.Second)

	nBytes, err := conn.WriteToUDP(body, ua)
	if err != nil {
		return err
	}
	log.Printf("wrote %d bytes of response back to %s\n", nBytes, ua)
	return nil
}
