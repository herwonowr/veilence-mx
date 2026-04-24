package audit

import "context"

type correlationKey string
type ipAddressKey string
type userAgentKey string

const (
	ctxCorrelationID correlationKey = "correlation_id"
	ctxIPAddress     ipAddressKey   = "ip_address"
	ctxUserAgent     userAgentKey   = "user_agent"
)

// WithCorrelationID returns a new context with the correlation ID set.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxCorrelationID, id)
}

// CorrelationIDFromContext extracts the correlation ID from the request context.
// Returns an empty string if not set.
func CorrelationIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxCorrelationID).(string); ok {
		return v
	}
	return ""
}

// WithIPAddress returns a new context with the IP address set.
func WithIPAddress(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, ctxIPAddress, ip)
}

// IPAddressFromContext extracts the IP address from the request context.
// Returns an empty string if not set.
func IPAddressFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxIPAddress).(string); ok {
		return v
	}
	return ""
}

// WithUserAgent returns a new context with the user agent set.
func WithUserAgent(ctx context.Context, ua string) context.Context {
	return context.WithValue(ctx, ctxUserAgent, ua)
}

// UserAgentFromContext extracts the user agent from the request context.
// Returns an empty string if not set.
func UserAgentFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxUserAgent).(string); ok {
		return v
	}
	return ""
}
