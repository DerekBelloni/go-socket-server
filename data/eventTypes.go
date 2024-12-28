package data

type NostrEventMessage interface {
	EventType() string
}

type EventMessage struct {
	Event      NostrEventMessage `json:",inline"`
	UserPubkey *string
	UUID       *string
}

type FollowsEvent struct {
	Data []interface{} `json:"follows"`
}

func (FollowsEvent) EventType() string {
	return "follows"
}

type UserEvent struct {
	Data []interface{} `json:"user"`
}

// probably dont need this return with the inline json
func (UserEvent) EventType() string {
	return "user"
}
