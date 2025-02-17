package tracking

type TrackerManager struct {
	EmbeddedTracker Tracker[EmbeddedMetadata]
}

func NewTrackerManager() *TrackerManager {
	return &TrackerManager{
		EmbeddedTracker: NewEmbeddedTracker(),
	}
}
