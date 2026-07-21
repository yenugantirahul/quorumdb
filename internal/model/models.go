package model

type PutRequest struct {
	Value string `json:"value"`
}

type Record struct {
	Value   string `json:"value"`
	Version uint64 `json:"version"`
}

type Resposnse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Error string `json:"error"`
}
