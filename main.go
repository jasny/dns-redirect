package main

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/acme/autocert"
)

var (
	redirectDomains map[string]int
	useCache        bool
)

func init() {
	initRedirectDomains()
	initDNS()
	initStatic()
}

func initRedirectDomains() {
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

func handler(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	// When visiting a redirect domain directly, serve the static page
	if _, exists := redirectDomains[host]; exists {
		serveStaticPage(w, r)
		return
	}

	// Fetch the CNAME record for the host
	cname, err := getCNAME(host)
	if err != nil {
		cname, err = getCNAME("@." + host)
	}

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

func startHttpServer(certManager *autocert.Manager) {
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "80"
	}

	// Use the certManager's HTTPHandler to handle ACME challenges for Let's Encrypt
	log.Printf("Starting HTTP server on :%s", httpPort)
	log.Fatal(http.ListenAndServe(":"+httpPort, certManager.HTTPHandler(http.HandlerFunc(handler))))
}

func startHttpsServer(certManager *autocert.Manager) {
	httpPort := os.Getenv("HTTP_PORT")
	httpsPort := os.Getenv("HTTPS_PORT")

	if httpsPort == "" {
		if httpPort != "" && httpPort != "80" {
			return
		}
		httpsPort = "443"
	}

	if httpPort != "" && httpPort != "80" {
		log.Fatal("Unable to serve HTTPS traffic if HTTP is not served on port 80 due to Let's Encrypt challenge")
	}

	// HTTPS server with Let's Encrypt
	server := &http.Server{
		Addr:    ":" + httpsPort,
		Handler: http.HandlerFunc(handler),
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	// Start the HTTPS server
	log.Printf("Starting HTTPS server on :%s", httpsPort)
	log.Fatal(server.ListenAndServeTLS("", ""))
}

func main() {
	// Create an autocert manager for Let's Encrypt certificates
	certManager := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache("certs"), // Folder for storing certificates
	}

	go startHttpServer(&certManager)
	go startHttpsServer(&certManager)
	select {}
}
