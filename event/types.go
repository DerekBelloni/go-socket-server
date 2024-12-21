package event

type NostrEvent interface {
	EventType() string
}

type EventMessage struct {
	Event      NostrEvent
	UserPubkey *string
	UUID       *string
}

type FollowsEvent struct {
	Event []interface{}
}

func (FollowsEvent) EventType() string {
	return "follows"
}

type UserEvent struct {
	Event []interface{}
}

func (UserEvent) EventType() string {
	return "user"
}
