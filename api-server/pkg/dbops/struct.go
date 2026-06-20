package dbops

type DistinctItem struct {
	Value string `json:"value,omitempty"`
	Count int    `json:"count"`
}
