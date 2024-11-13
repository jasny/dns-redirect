package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

var (
	clientConfig *dns.ClientConfig
	dnsClient    *dns.Client
	dnsCache     = make(map[string]cachedCNAME)
	dnsCacheLock sync.RWMutex
)

type cachedCNAME struct {
	cname  string
	expiry time.Time
}

func initDNS() {
	// Initialize DNS client configuration
	configFile := os.Getenv("DNS_CONFIG_FILE")
	if configFile == "" {
		configFile = "/etc/resolv.conf"
	}

	var err error
	clientConfig, err = dns.ClientConfigFromFile(configFile)
	if err != nil {
		// If there's an error reading the config file, fall back to Google Public DNS
		log.Printf("Error reading DNS config, defaulting to Google Public DNS: %v", err)
		clientConfig = &dns.ClientConfig{
			Servers: []string{"8.8.8.8"},
			Port:    "53",
		}
	}

	// Set up a new DNS client
	dnsClient = new(dns.Client)

	// Initialize the useCache flag based on CACHE_DNS environment variable
	cacheEnv := strings.ToLower(os.Getenv("CACHE_DNS"))
	useCache = cacheEnv != "" && cacheEnv != "false" && cacheEnv != "off" && cacheEnv != "0"

	if useCache {
		log.Print("Using DNS cache")
	}
}

func getCNAME(host string) (string, error) {
	if useCache {
		// Check cache for the CNAME
		dnsCacheLock.RLock()
		if cached, found := dnsCache[host]; found {
			if time.Now().Before(cached.expiry) {
				dnsCacheLock.RUnlock()
				return cached.cname, nil
			}
			// Cache expired, need to refresh
			dnsCacheLock.RUnlock()
		} else {
			dnsCacheLock.RUnlock()
		}
	}

	// Set up a new DNS message
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(host), dns.TypeCNAME)

	// Execute the query to a public DNS server
	response, _, err := dnsClient.Exchange(msg, clientConfig.Servers[0]+":"+clientConfig.Port)
	if err != nil {
		return "", err
	}

	// Loop through the answers and retrieve the CNAME
	for _, ans := range response.Answer {
		if cnameRecord, ok := ans.(*dns.CNAME); ok {
			if useCache {
				// Cache the CNAME with the TTL from the DNS response
				ttl := time.Duration(cnameRecord.Hdr.Ttl) * time.Second
				expiry := time.Now().Add(ttl)

				dnsCacheLock.Lock()
				dnsCache[host] = cachedCNAME{cname: cnameRecord.Target, expiry: expiry}
				dnsCacheLock.Unlock()
			}
			return cnameRecord.Target, nil
		}
	}

	return "", fmt.Errorf("no CNAME record found for host: %s", host)
}
