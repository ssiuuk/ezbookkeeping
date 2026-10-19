package core

import "context"

// DetachedContext represents a context for background work started by another context,
// it keeps the context id, client ip and client locale of the original context, but is not canceled when the original context is done
type DetachedContext struct {
	context.Context
	contextId    string
	clientIP     string
	clientLocale string
}

// ClientIP returns the client IP address of the original context
func (c *DetachedContext) ClientIP() string {
	return c.clientIP
}

// GetContextId returns the context id of the original context
func (c *DetachedContext) GetContextId() string {
	return c.contextId
}

// GetClientLocale returns the client locale name of the original context
func (c *DetachedContext) GetClientLocale() string {
	return c.clientLocale
}

// NewDetachedContext returns a new detached context of the given context
func NewDetachedContext(c Context) *DetachedContext {
	return &DetachedContext{
		Context:      context.Background(),
		contextId:    c.GetContextId(),
		clientIP:     c.ClientIP(),
		clientLocale: c.GetClientLocale(),
	}
}
