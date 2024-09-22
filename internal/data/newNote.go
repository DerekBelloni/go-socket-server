package data

type NewNote struct {
	PrivHexKey string `json:"privHexKey"`
	PubHexKey  string `json:"pubHexKey"`
	Kind       int    `json:"kind"`
	Content    string `json:"content"`
}
