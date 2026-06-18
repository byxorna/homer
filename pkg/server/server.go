package server

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/byxorna/homer/internal/version"
	"github.com/byxorna/homer/pkg/types"
	"github.com/miekg/dns"
)

type srv struct {
	cfg       Config
	mux       *http.ServeMux
	dnsClient dns.Client
	indexPage []byte
}

//go:embed index.html
var indexHTML string

var indexTmpl = template.Must(template.New("index").Parse(indexHTML))

// Server is the server interface
type Server interface {
	ListenAndServe() error
}

// NewServer does what it says
func NewServer(cfg Config) (Server, error) {
	var indexBuf bytes.Buffer
	err := indexTmpl.Execute(&indexBuf, struct {
		Version   string
		BuildDate string
		Package   string
		StartTime string
	}{
		Version:   version.Version,
		BuildDate: version.BuildDate,
		Package:   version.Package,
		StartTime: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to render index page: %v", err)
	}

	server := srv{
		cfg:       cfg,
		mux:       http.NewServeMux(),
		dnsClient: dns.Client{},
		indexPage: indexBuf.Bytes(),
	}
	server.mux.Handle("/.well-known/dns-query", &server)
	server.mux.HandleFunc("/", server.serveIndex)
	return &server, nil
}

func (s *srv) serveIndex(res http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(res, req)
		return
	}
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	res.Write(s.indexPage)
}

func (s *srv) ListenAndServe() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.ListenAddr, s.cfg.ListenPort)
	log.Printf("Listening on %s\n", addr)
	return http.ListenAndServe(addr, s.mux)
}

func (s *srv) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if s.cfg.Debug {
		dumpreq, err := httputil.DumpRequest(req, true)
		if err != nil {
			log.Printf("bro... cant dump request: %v\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Printf("got request for dns-query: %s\n", bytes.NewBuffer(dumpreq).String())
	}
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
			log.Printf("error decoding body parameter: %s -> %v\n", encodedBodyParam, err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		// parse []byte into a dns RR
		var msg dns.Msg
		err = msg.Unpack(dnsReq)
		if err != nil {
			log.Printf("error unpacking request into DNS message: %v\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		//log.Printf("got message: %s\n", msg.String())
		if len(msg.Question) < 1 {
			log.Printf("no questions in message\n")
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		// fire off a request
		if s.cfg.Debug {
			log.Printf("Sending DNS request...\n")
		}
		resolver := s.cfg.Resolvers[0]
		dnsResp, rtt, err := s.dnsClient.Exchange(&msg, fmt.Sprintf("%s:%d", resolver, 53))
		if err != nil {
			log.Printf("error exchanging DNS request with %s: %v\n", resolver, err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if s.cfg.Debug {
			log.Printf("Response Message from %s in %s:\n%s", resolver, rtt.String(), dnsResp)
		}

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
	//TODO implement POST DOH
	res.WriteHeader(http.StatusMethodNotAllowed)
	return
}
