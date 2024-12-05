package data

type NewNote struct {
	PrivHexKey string  `json:"privHexKey"`
	PubHexKey  string  `json:"pubHexKey"`
	Kind       float64 `json:"kind"`
	Content    string  `json:"content"`
}
