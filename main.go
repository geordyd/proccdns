package main

import (
	"flag"
	"log"
	"strings"

	"github.com/miekg/dns"
)

type DNSProxy struct {
	server         *dns.Server
	dnsServers     []string
	domainMappings map[string]string
}

func NewDNSProxy(addr string, dnsServers []string, domainMappings map[string]string) *DNSProxy {
	return &DNSProxy{
		server: &dns.Server{
			Addr: addr,
			Net:  "udp",
		},
		dnsServers:     dnsServers,
		domainMappings: domainMappings,
	}
}

func (p *DNSProxy) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	// Log the incoming DNS query
	queryName := r.Question[0].Name
	log.Printf("Received DNS query for %s from %s", queryName, w.RemoteAddr())

	// Check if this is a mapped domain
	queryNameLower := strings.ToLower(queryName)
	for suffix, ip := range p.domainMappings {
		// Remove trailing dot from query name for comparison
		queryNameWithoutDot := strings.TrimSuffix(queryNameLower, ".")
		if strings.HasSuffix(queryNameWithoutDot, strings.TrimSuffix(suffix, ".")) {
			log.Printf("MAPPED DOMAIN: %s -> %s", queryName, ip)
			m := new(dns.Msg)
			m.SetReply(r)

			// Create an A record response
			rr, err := dns.NewRR(queryName + " 60 IN A " + ip)
			if err != nil {
				log.Printf("Error creating RR: %v", err)
				m.SetRcode(r, dns.RcodeServerFailure)
				w.WriteMsg(m)
				return
			}
			m.Answer = append(m.Answer, rr)
			w.WriteMsg(m)
			return
		}
	}

	// Create a DNS client
	client := new(dns.Client)

	// Try each DNS server in order
	var lastErr error
	for i, server := range p.dnsServers {
		resp, _, err := client.Exchange(r, server+":53")
		if err != nil {
			lastErr = err
			log.Printf("DNS SERVER %d FAILED (%s): %v, trying next", i+1, server, err)
			continue
		}
		log.Printf("RESOLVED BY SERVER %d: %s -> %s", i+1, queryName, server)
		w.WriteMsg(resp)
		return
	}

	// If we get here, all servers failed
	log.Printf("ALL DNS SERVERS FAILED. Last error: %v", lastErr)
	m := new(dns.Msg)
	m.SetReply(r)
	m.SetRcode(r, dns.RcodeServerFailure)
	w.WriteMsg(m)
}

func (p *DNSProxy) Start() error {
	// Set up the DNS handler
	dns.HandleFunc(".", p.handleDNS)

	// Start the server
	log.Printf("Starting DNS proxy on %s", p.server.Addr)
	log.Printf("DNS servers (in priority order):")
	for i, server := range p.dnsServers {
		log.Printf("  %d. %s", i+1, server)
	}
	log.Printf("Domain mappings:")
	for suffix, ip := range p.domainMappings {
		log.Printf("  %s -> %s", suffix, ip)
	}
	return p.server.ListenAndServe()
}

func main() {
	// Define command line flags
	listenAddr := flag.String("listen", ":53", "Address to listen on")
	dnsServers := flag.String("servers", "8.8.8.8,1.1.1.1", "Comma-separated list of DNS servers in priority order")
	domainMappings := flag.String("domains", "", "Comma-separated list of domain=ip mappings (e.g., '.docker=172.168.1.1,.test=192.168.1.1')")
	flag.Parse()

	// Parse DNS servers
	servers := strings.Split(*dnsServers, ",")
	for i := range servers {
		servers[i] = strings.TrimSpace(servers[i])
	}

	// Parse domain mappings
	mappings := make(map[string]string)
	if *domainMappings != "" {
		for _, mapping := range strings.Split(*domainMappings, ",") {
			parts := strings.Split(mapping, "=")
			if len(parts) == 2 {
				domain := parts[0]
				ip := parts[1]
				// Ensure domain starts with a dot
				if !strings.HasPrefix(domain, ".") {
					domain = "." + domain
				}
				mappings[domain] = ip
			}
		}
	}

	// Create a new DNS proxy with configurable servers
	proxy := NewDNSProxy(
		*listenAddr,
		servers,
		mappings,
	)

	// Start the proxy
	if err := proxy.Start(); err != nil {
		log.Fatalf("Failed to start DNS proxy: %v", err)
	}
}
