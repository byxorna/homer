# homer
DNS over HTTPS implementation (IETF, _not_ google implementation)

# Client

Client listens on `53/udp` for DNS messages. It encodes them into HTTP requests and sends them to the server. It also decodes responses via HTTP and writes the decoded UDP response back to the DNS resolver.

# Server

Server listens on HTTP[S] for DNS over HTTP requests, decodes them, performs resolution, and returns the response.

# Links

* https://github.com/pforemski/dingo
* https://tools.ietf.org/html/draft-ietf-doh-dns-over-https-02
* https://developers.google.com/speed/public-dns/docs/dns-over-https
* https://datatracker.ietf.org/wg/doh/about/
* https://github.com/m13253/dns-over-https
