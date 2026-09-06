// Package ledger is the core of this entire project — a real double-entry
// accounting engine. Every other package (wallet, gateway, escrow) is
// built ON TOP of this one; none of them ever mutate a balance directly.
package ledger

import (
	"strconv"
	"sync"
	"time"
)

// AuditEntry and AuditLog implement the guide's Audit Logs section:
// append-only, never edited or deleted, because the entire point is being
// able to trust it represents exactly what happened.
type AuditEntry struct {
	ID        string
	Timestamp time.Time
	Actor     string
	Action    string
	Target    string
	Metadata  map[string]any
}

type AuditLog struct {
	mu      sync.Mutex
	entries []AuditEntry
	nextID  int
}

func NewAuditLog() *AuditLog {
	return &AuditLog{nextID: 1}
}

// Record is the ONLY way to add to the log — there is deliberately no
// Update or Delete method anywhere in this type.
func (l *AuditLog) Record(actor, action, target string, metadata map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, AuditEntry{
		ID:        generateAuditID(l.nextID),
		Timestamp: time.Now(),
		Actor:     actor,
		Action:    action,
		Target:    target,
		Metadata:  metadata,
	})
	l.nextID++
}

func (l *AuditLog) All() []AuditEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]AuditEntry, len(l.entries))
	copy(out, l.entries) // a COPY — callers can never mutate the real log through this
	return out
}

// ForTarget filters the log to entries about one specific resource — the
// "what happened to THIS account, ever?" query a support/compliance team
// would actually run.
func (l *AuditLog) ForTarget(target string) []AuditEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []AuditEntry
	for _, e := range l.entries {
		if e.Target == target {
			out = append(out, e)
		}
	}
	return out
}

func generateAuditID(n int) string {
	return "audit_" + strconv.Itoa(n)
}
