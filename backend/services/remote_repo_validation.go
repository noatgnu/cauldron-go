package services

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func ValidateRemoteRepoURL(repoURL string, allowedHosts []string, restrictToAllowedHosts bool) error {
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("invalid repository URL")
	}

	if parsed.Scheme != "https" {
		return fmt.Errorf("repository URL must use https://")
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("repository URL has no host")
	}

	for _, allowed := range allowedHosts {
		if strings.EqualFold(host, allowed) {
			return nil
		}
	}

	if restrictToAllowedHosts {
		return fmt.Errorf("repository host is not on the allowed hosts list")
	}

	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("could not resolve repository host")
	}

	for _, ip := range ips {
		if isDisallowedTargetIP(ip) {
			return fmt.Errorf("repository host resolves to a disallowed address")
		}
	}

	return nil
}

func isDisallowedTargetIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified()
}
