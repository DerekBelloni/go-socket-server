package tracking

type TrackerManager struct {
	EmbeddedTracker     Tracker[EmbeddedMetadata]
	NPubMetadataTracker Tracker[NPubMetadata]
}

func NewTrackerManager() *TrackerManager {
	return &TrackerManager{
		EmbeddedTracker:     NewEmbeddedTracker(),
		NPubMetadataTracker: NewNPubMetadataTracker(),
	}
}
