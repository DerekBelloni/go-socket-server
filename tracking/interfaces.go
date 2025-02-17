package tracking

type UserContext struct {
	ID   string
	Type string
}

type SubscriptionMetadata interface {
	GetUserContext() UserContext
}

type Tracker[T SubscriptionMetadata] interface {
	Track(subscriptionID string, metadata T) error
	Lookup(event []interface{}) (T, error)
}
