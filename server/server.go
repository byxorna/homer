package server

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/miekg/dns"

	"github.com/byxorna/homer/types"
)

type srv struct {
	cfg Config
	mux *http.ServeMux
	//udpConn   *net.UDPConn
	dnsClient dns.Client
}

// Server is the server interface
type Server interface {
	ListenAndServe() error
}

// NewServer does what it says
func NewServer(cfg Config) (Server, error) {
	//resolverAddr := net.UDPAddr{
	//	IP:   net.ParseIP(cfg.Resolvers[0]),
	//	Port: 53,
	//}
	//log.Printf("establishing a connection to %v\n", resolverAddr)
	//udpConn, err := net.DialUDP("udp", nil, &resolverAddr)
	//if err != nil {
	//	return nil, err
	//}
	server := srv{
		cfg: cfg,
		mux: http.NewServeMux(),
		//udpConn:   udpConn,
		dnsClient: dns.Client{},
	}
	server.mux.Handle("/.well-known/dns-query", &server)
	return &server, nil
}

func (s *srv) ListenAndServe() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.ListenAddr, s.cfg.ListenPort)
	log.Printf("Listening on %s\n", addr)
	return http.ListenAndServe(addr, s.mux)
}

func (s *srv) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	dumpreq, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Printf("bro... cant dump request: %v\n", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("got request for dns-query: %s\n", bytes.NewBuffer(dumpreq).String())
	if req.Method == http.MethodGet {
		ctype := req.Header.Get("content-type")
		if ctype != types.TypeDNSUDPWireFormat {
			res.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		res.Header().Set("content-type", types.TypeDNSUDPWireFormat)

		qparams := req.URL.Query()
		log.Printf("req URL: %v\n", req.URL)
		ct := qparams.Get("ct")
		if ct != "" && ct != types.TypeDNSUDPWireFormat {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		// decode body
		encodedBodyParam := qparams.Get("body")
		if encodedBodyParam == "" {
			log.Printf("body parameter missing\n")
			res.WriteHeader(http.StatusExpectationFailed)
			return
		}

		dnsReq, err := base64.URLEncoding.DecodeString(encodedBodyParam)
		if err != nil {
			log.Printf("unable to decode body parameter: %s -> %v\n", encodedBodyParam, err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		// parse []byte into a dns RR
		var msg dns.Msg
		err = msg.Unpack(dnsReq)
		if err != nil {
			log.Printf("unable to unpack request into DNS message: %v\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Printf("got message: %s\n", msg.String())
		if len(msg.Question) < 1 {
			log.Printf("no questions in message\n")
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		// fire off a request
		log.Printf("Sending DNS request...\n")
		dnsResp, rtt, err := s.dnsClient.Exchange(&msg, fmt.Sprintf("%s:%d", s.cfg.Resolvers[0], 53))
		log.Printf("Response in %s\n", rtt.String())
		if err != nil {
			log.Printf("error: %v\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Printf("Response Message: %s\n", dnsResp)

		packedMsg, err := dnsResp.Pack()
		if err != nil {
			log.Printf("error packing DNS response: %v\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = res.Write(packedMsg)
		if err != nil {
			log.Printf("error writing DNS response to response body: %v\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}
	//TODO implement POST
	res.WriteHeader(http.StatusMethodNotAllowed)
	return
}
