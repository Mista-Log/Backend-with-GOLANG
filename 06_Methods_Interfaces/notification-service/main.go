// Project 2: Notification Service
//
// Sends notifications through any number of channels behind one Notifier
// interface. The interesting piece is MultiNotifier: it ITSELF satisfies
// Notifier by fanning out to a slice of other Notifiers — the "Composite"
// pattern, which interfaces make almost free in Go. Also covers interface
// embedding (ValidatedNotifier), a type switch for per-channel formatting,
// and `any` for freeform event payloads.
package main

import (
	"fmt"
)

// Notifier is the one method every channel needs.
type Notifier interface {
	Send(message string) error
}

// Validator is a small, separate interface — not every Notifier needs
// pre-send validation, so it isn't folded into Notifier itself.
type Validator interface {
	Validate() error
}

// ValidatedNotifier is COMPOSED from the two smaller interfaces above —
// interface embedding, same idea as the guide's ReadWriter example.
type ValidatedNotifier interface {
	Notifier
	Validator
}

// --- Email -----------------------------------------------------

type EmailNotifier struct {
	Address string
}

func (e EmailNotifier) Validate() error {
	if !containsAt(e.Address) {
		return fmt.Errorf("invalid email address %q", e.Address)
	}
	return nil
}

func (e EmailNotifier) Send(message string) error {
	fmt.Printf("  [EMAIL to %s] %s\n", e.Address, message)
	return nil
}

func containsAt(s string) bool {
	for _, r := range s {
		if r == '@' {
			return true
		}
	}
	return false
}

// --- SMS -----------------------------------------------------

const smsCharLimit = 160

type SMSNotifier struct {
	PhoneNumber string
}

func (s SMSNotifier) Validate() error {
	if len(s.PhoneNumber) < 7 {
		return fmt.Errorf("phone number %q looks too short", s.PhoneNumber)
	}
	return nil
}

func (s SMSNotifier) Send(message string) error {
	if len(message) > smsCharLimit {
		message = message[:smsCharLimit-3] + "..."
	}
	fmt.Printf("  [SMS to %s] %s\n", s.PhoneNumber, message)
	return nil
}

// --- Push -----------------------------------------------------

// PushNotifier deliberately has NO Validate method — it satisfies Notifier
// but not ValidatedNotifier, mirroring the CryptoProcessor/Refundable split
// from the Payment Gateway project.
type PushNotifier struct {
	DeviceToken string
}

func (p PushNotifier) Send(message string) error {
	fmt.Printf("  [PUSH to device %s] %s\n", p.DeviceToken, message)
	return nil
}

// --- MultiNotifier (the Composite pattern) -----------------------------

// MultiNotifier holds a slice of Notifiers and ITSELF satisfies Notifier,
// because it has a Send method too. Callers can hand a MultiNotifier
// anywhere a single Notifier is expected — including nesting a MultiNotifier
// inside ANOTHER MultiNotifier, since it's just another Notifier as far as
// the type system is concerned.
type MultiNotifier struct {
	channels []Notifier
}

func NewMultiNotifier(channels ...Notifier) *MultiNotifier {
	return &MultiNotifier{channels: channels}
}

// Send fans the message out to every channel, validating first wherever a
// channel happens to ALSO satisfy Validator — a type assertion decides that
// per-channel, at runtime.
func (m *MultiNotifier) Send(message string) error {
	var failures []string
	for _, ch := range m.channels {
		if v, ok := ch.(Validator); ok { // does THIS channel support validation?
			if err := v.Validate(); err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", describe(ch), err))
				continue
			}
		}
		if err := ch.Send(message); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", describe(ch), err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%d channel(s) failed: %v", len(failures), failures)
	}
	return nil
}

// describe uses a TYPE SWITCH — distinct from a type ASSERTION — to produce
// a human-readable label per concrete Notifier type, purely for logging.
func describe(n Notifier) string {
	switch v := n.(type) {
	case EmailNotifier:
		return "email(" + v.Address + ")"
	case SMSNotifier:
		return "sms(" + v.PhoneNumber + ")"
	case PushNotifier:
		return "push(" + v.DeviceToken + ")"
	case *MultiNotifier:
		return fmt.Sprintf("multi(%d channels)", len(v.channels))
	default:
		return "unknown channel"
	}
}

// --- Event log with freeform payloads -----------------------------

// NotificationEvent uses `any` for Payload since different triggers carry
// completely different extra data (an order ID, a user ID, a raw amount...).
type NotificationEvent struct {
	Trigger string
	Payload any
}

func logEvent(e NotificationEvent) {
	fmt.Printf("Event: %-20s payload: %v (%T)\n", e.Trigger, e.Payload, e.Payload)
}

func main() {
	email := EmailNotifier{Address: "user@example.com"}
	sms := SMSNotifier{PhoneNumber: "+15551234567"}
	push := PushNotifier{DeviceToken: "abc123"}
	badEmail := EmailNotifier{Address: "not-an-email"}

	fmt.Println("=== Single channel ===")
	email.Send("Your order has shipped!")

	fmt.Println("\n=== MultiNotifier fanning out to all three ===")
	all := NewMultiNotifier(email, sms, push)
	if err := all.Send("Reminder: your appointment is tomorrow."); err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Println("\n=== MultiNotifier with a channel that fails validation ===")
	withBadChannel := NewMultiNotifier(email, badEmail, sms)
	if err := withBadChannel.Send("This will partially fail."); err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Println("\n=== Nested MultiNotifier (a Notifier made of Notifiers) ===")
	criticalAlerts := NewMultiNotifier(sms, push)     // urgent channels only
	everything := NewMultiNotifier(email, criticalAlerts) // email + the urgent bundle
	everything.Send("System maintenance starting now.")

	fmt.Println("\n=== Long SMS gets truncated by SMSNotifier.Send itself ===")
	sms.Send("This is a deliberately very long message that exceeds the SMS character limit, so SMSNotifier's own Send method should truncate it before printing, ensuring the receipt reflects what an actual carrier would allow through, roughly.")

	fmt.Println("\n=== Freeform event payloads via `any` ===")
	logEvent(NotificationEvent{Trigger: "order.shipped", Payload: 48213})
	logEvent(NotificationEvent{Trigger: "user.signup", Payload: map[string]string{"email": "new@example.com"}})
	logEvent(NotificationEvent{Trigger: "payment.failed", Payload: 19.99})
}
