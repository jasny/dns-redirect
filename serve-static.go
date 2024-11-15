package main

import (
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
)

var (
	tmpl             *template.Template
	baseTemplateData TemplateData
)

type TemplateData struct {
	IpAddress         string
	RedirectDomain    string
	RedirectDomain301 string
	RedirectDomain302 string
	RedirectDomain303 string
	RedirectDomain307 string
	RedirectDomain308 string
}

func initStatic() {
	// Parse the template
	var err error
	tmpl, err = template.ParseFiles("static/index.html")
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	// Get the IP address using the first redirect domain that is set
	redirectDomain := ""
	for _, domain := range []string{
		os.Getenv("REDIRECT_DOMAIN_301"),
		os.Getenv("REDIRECT_DOMAIN_302"),
		os.Getenv("REDIRECT_DOMAIN_303"),
		os.Getenv("REDIRECT_DOMAIN_307"),
		os.Getenv("REDIRECT_DOMAIN_308"),
	} {
		if domain != "" {
			redirectDomain = domain
			break
		}
	}
	ipAddress, ipErr := getIpAddress(redirectDomain)
	if ipErr != nil {
		log.Fatalf("Error getting IP address: %v", ipErr)
	}

	// Initialize the base template data
	baseTemplateData = TemplateData{
		IpAddress:         ipAddress,
		RedirectDomain301: os.Getenv("REDIRECT_DOMAIN_301"),
		RedirectDomain302: os.Getenv("REDIRECT_DOMAIN_302"),
		RedirectDomain303: os.Getenv("REDIRECT_DOMAIN_303"),
		RedirectDomain307: os.Getenv("REDIRECT_DOMAIN_307"),
		RedirectDomain308: os.Getenv("REDIRECT_DOMAIN_308"),
	}
}

func serveStaticPage(w http.ResponseWriter, r *http.Request) {
	host, _, hostErr := net.SplitHostPort(r.Host)
	if hostErr != nil {
		host = r.Host
	}

	data := baseTemplateData
	data.RedirectDomain = host

	// Serve the static page
	err := tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unexpected error", http.StatusInternalServerError)
	}
}
