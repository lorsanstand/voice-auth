package models

type VectorDBCreate struct {
	Vector   []float32
	Content  string
	Metadata map[string]string
}

type VectorDB struct {
	ID         string
	Vector     []float32
	Content    string
	Metadata   map[string]string
	Similarity float32
}
