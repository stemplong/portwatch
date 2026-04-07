// Package notifier provides webhook and email notification capabilities
// for portwatch, dispatching alerts when unexpected port changes are detected.
package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/user/portwatch/config"
)

// PortEvent represents a port change event to be dispatched.
type PortEvent struct {
	Hostname  string    `json:"hostname"`
	EventType string    `json:"event_type"` // "opened" or "closed"
	Port      uint16    `json:"port"`
	Protocol  string    `json:"protocol"`
	Timestamp time.Time `json:"timestamp"`
}

// Notifier dispatches alerts via configured channels.
type Notifier struct {
	cfg *config.Config
	client *http.Client
}

// New creates a new Notifier with the provided configuration.
func New(cfg *config.Config) *Notifier {
	return &Notifier{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send dispatches a PortEvent to all configured notification channels.
func (n *Notifier) Send(event PortEvent) error {
	var errs []string

	if n.cfg.Webhook.URL != "" {
		if err := n.sendWebhook(event); err != nil {
			errs = append(errs, fmt.Sprintf("webhook: %v", err))
		}
	}

	if n.cfg.Email.SMTPHost != "" && len(n.cfg.Email.To) > 0 {
		if err := n.sendEmail(event); err != nil {
			errs = append(errs, fmt.Sprintf("email: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// sendWebhook posts the event as JSON to the configured webhook URL.
func (n *Notifier) sendWebhook(event PortEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, n.cfg.Webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if n.cfg.Webhook.Secret != "" {
		req.Header.Set("X-Portwatch-Secret", n.cfg.Webhook.Secret)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

// sendEmail sends an alert email via SMTP for the given event.
func (n *Notifier) sendEmail(event PortEvent) error {
	subject := fmt.Sprintf("[portwatch] Port %s: %d/%s on %s",
		event.EventType, event.Port, event.Protocol, event.Hostname)

	body := fmt.Sprintf(
		"Portwatch Alert\n\nHost:      %s\nEvent:     %s\nPort:      %d\nProtocol:  %s\nTime:      %s\n",
		event.Hostname, event.EventType, event.Port, event.Protocol,
		event.Timestamp.Format(time.RFC1123),
	)

	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		n.cfg.Email.From,
		strings.Join(n.cfg.Email.To, ", "),
		subject,
		body,
	))

	addr := fmt.Sprintf("%s:%d", n.cfg.Email.SMTPHost, n.cfg.Email.SMTPPort)
	var auth smtp.Auth
	if n.cfg.Email.Username != "" {
		auth = smtp.PlainAuth("", n.cfg.Email.Username, n.cfg.Email.Password, n.cfg.Email.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, n.cfg.Email.From, n.cfg.Email.To, msg); err != nil {
		return fmt.Errorf("sending mail: %w", err)
	}
	return nil
}
