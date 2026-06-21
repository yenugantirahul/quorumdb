package model

type PutRequest struct {
	Value string `json:"value"`
}

type Resposnse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Error string `json:"error"`
}
