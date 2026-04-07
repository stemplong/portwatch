// Package alert provides webhook and email notification capabilities
// for portwatch when unexpected port changes are detected.
package alert

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

// PortEvent represents a change in port state that triggered an alert.
type PortEvent struct {
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"` // "opened" or "closed"
	Port      uint16    `json:"port"`
	Protocol  string    `json:"protocol"`
	LocalAddr string    `json:"local_addr"`
	PID       int       `json:"pid,omitempty"`
	Process   string    `json:"process,omitempty"`
}

// Notifier dispatches alerts via configured channels.
type Notifier struct {
	cfg *config.Config
	client *http.Client
}

// New creates a new Notifier using the provided configuration.
func New(cfg *config.Config) *Notifier {
	return &Notifier{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send dispatches an alert for the given port event through all
// configured notification channels (webhook, email).
func (n *Notifier) Send(event PortEvent) error {
	var errs []string

	if n.cfg.Webhook.URL != "" {
		if err := n.sendWebhook(event); err != nil {
			errs = append(errs, fmt.Sprintf("webhook: %v", err))
		}
	}

	if n.cfg.Email.To != "" && n.cfg.Email.SMTPHost != "" {
		if err := n.sendEmail(event); err != nil {
			errs = append(errs, fmt.Sprintf("email: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("alert delivery errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// sendWebhook posts a JSON-encoded PortEvent to the configured webhook URL.
func (n *Notifier) sendWebhook(event PortEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshalling event: %w", err)
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

// sendEmail sends a plain-text alert email via SMTP.
func (n *Notifier) sendEmail(event PortEvent) error {
	subject := fmt.Sprintf("[portwatch] Port %s: %d/%s",
		strings.ToUpper(event.EventType), event.Port, event.Protocol)

	body := fmt.Sprintf(
		"Port change detected on %s\n\nEvent : %s\nPort  : %d/%s\nAddr  : %s\nTime  : %s\n",
		n.cfg.Hostname,
		strings.ToUpper(event.EventType),
		event.Port,
		event.Protocol,
		event.LocalAddr,
		event.Timestamp.Format(time.RFC1123),
	)
	if event.Process != "" {
		body += fmt.Sprintf("Process: %s (PID %d)\n", event.Process, event.PID)
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		n.cfg.Email.From, n.cfg.Email.To, subject, body)

	addr := fmt.Sprintf("%s:%d", n.cfg.Email.SMTPHost, n.cfg.Email.SMTPPort)
	var auth smtp.Auth
	if n.cfg.Email.Username != "" {
		auth = smtp.PlainAuth("", n.cfg.Email.Username, n.cfg.Email.Password, n.cfg.Email.SMTPHost)
	}

	return smtp.SendMail(addr, auth, n.cfg.Email.From, []string{n.cfg.Email.To}, []byte(msg))
}
