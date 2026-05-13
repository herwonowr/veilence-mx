package entity

import "strings"

// EmailDomainAllowed checks if an email's domain is in the allowed list.
// Returns true if allowedDomains is empty (no restriction).
// Uses exact domain match (no subdomain matching).
// Normalizes: case-insensitive, trims whitespace, strips trailing DNS dots.
func EmailDomainAllowed(email string, allowedDomains []string) bool {
	if len(allowedDomains) == 0 {
		return true
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || parts[1] == "" {
		return false
	}
	domain := NormalizeDomain(parts[1])
	for _, d := range allowedDomains {
		if NormalizeDomain(d) == domain {
			return true
		}
	}
	return false
}

// NormalizeDomain lowercases, trims whitespace, and strips trailing DNS dots.
func NormalizeDomain(d string) string {
	return strings.ToLower(strings.TrimRight(strings.TrimSpace(d), "."))
}
