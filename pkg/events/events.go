// NATS subject constants and message envelope shared by all services.

package events

// Subject constants
const (
	// Auth domain
	SubjectUserCreated = "user.created"
	SubjectUserDeleted = "user.deleted"

	// Chat domain
	SubjectMessageCreated = "chat.message.created"
	SubjectMessageUpdated = "chat.message.updated"
	SubjectMessageDeleted = "chat.message.deleted"

	// Presence domain
	SubjectUserOnline  = "presence.user.online"
	SubjectUserOffline = "presence.user.offline"

	// User domain
	SubjectProfileUpdated = "user.profile.updated"
)

// Envelope wraps every NATS message published on the bus.
// All consumers must be able to decode this shape before
// decoding the service-specific Payload.
type Envelope[T any] struct {
	Subject   string `json:"subject"`
	Timestamp int64  `json:"timestamp"` // Unix milliseconds (UTC)
	TraceID   string `json:"trace_id,omitempty"`
	Payload   T      `json:"payload"`
}
