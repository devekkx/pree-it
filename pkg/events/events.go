package events

// NATS subject constants shared across all services.
const (
	UserCreated    = "user.created"
	UserDeleted    = "user.deleted"
	MessageCreated = "chat.message.created"
	MessageUpdated = "chat.message.updated"
	MessageDeleted = "chat.message.deleted"
	UserOnline     = "presence.user.online"
	UserOffline    = "presence.user.offline"
	ProfileUpdated = "user.profile.updated"
)

// Envelope wraps every NATS message for consistent structure.
type Envelope struct {
	Subject   string      `json:"subject"`
	Timestamp int64       `json:"timestamp"` // Unix milliseconds
	Payload   interface{} `json:"payload"`
}
