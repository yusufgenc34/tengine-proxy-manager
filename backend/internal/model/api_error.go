package model

type APIError struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code"`
}
