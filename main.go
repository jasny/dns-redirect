package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/miekg/dns"
)

var (
	clientConfig    *dns.ClientConfig
	dnsClient       *dns.Client
	redirectDomains map[string]int
)

func init() {
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

	// Initialize the redirect domains map from environment variables
	redirectDomains = make(map[string]int)
	domainEnvToStatus := map[string]int{
		"REDIRECT_DOMAIN_301": http.StatusMovedPermanently,
		"REDIRECT_DOMAIN_302": http.StatusFound,
		"REDIRECT_DOMAIN_303": http.StatusSeeOther,
		"REDIRECT_DOMAIN_307": http.StatusTemporaryRedirect,
		"REDIRECT_DOMAIN_308": http.StatusPermanentRedirect,
	}

	for envVar, status := range domainEnvToStatus {
		suffix := os.Getenv(envVar)
		if suffix != "" {
			redirectDomains[suffix] = status
			log.Printf("Redirect domain suffix: %s, Status code: %d", suffix, status)
		}
	}

	if len(redirectDomains) == 0 {
		log.Fatal("No redirect domains defined. Please set at least one REDIRECT_DOMAIN_ environment variable.")
	}
}

func getCNAME(host string) (string, error) {
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
			return cnameRecord.Target, nil
		}
	}

	return "", fmt.Errorf("no CNAME record found for host: %s", host)
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Split the host and port, if present
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		// If no port is specified, assume host is already just the hostname
		host = r.Host
	}

	// Fetch the CNAME record for the host
	cname, err := getCNAME(host)
	if err != nil {
		// If there is an error, log it and return an error to the user
		log.Printf("Error fetching CNAME for %s: %v", host, err)
		http.Error(w, "Unable to fetch CNAME", http.StatusInternalServerError)
		return
	}

	// Remove the trailing dot from the FQDN
	cname = cname[:len(cname)-1]

	// Determine the appropriate redirect domain and status code
	var targetDomain string
	var statusCode int
	for suffix, code := range redirectDomains {
		if strings.HasSuffix(cname, "."+suffix) {
			// Remove the redirect domain suffix from the CNAME
			targetDomain = strings.TrimSuffix(cname, "."+suffix)
			statusCode = code
			break
		}
	}

	if targetDomain == "" {
		log.Printf("Unexpected CNAME format or no matching redirect domain: %s", cname)
		http.Error(w, "Unexpected CNAME format", http.StatusInternalServerError)
		return
	}

	// Determine the scheme (http or https) based on the request
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	// Construct the target URL
	target := scheme + "://" + targetDomain + r.URL.String()

	// Redirect to the target with the appropriate status code
	log.Printf("Redirect (%d) %s to %s", statusCode, scheme+"://"+r.Host+r.URL.String(), target)
	http.Redirect(w, r, target, statusCode)
}

func main() {
	// Start the server
	http.HandleFunc("/", handler)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
