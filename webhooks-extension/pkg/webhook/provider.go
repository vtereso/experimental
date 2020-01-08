package webhook

// GitProvider is a type that can create and delete git repository webhooks
type GitProvider interface {
	AddWebhook() error
	DeleteWebhook() error
}
