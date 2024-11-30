package Server

import (
	"encoding/json"
	"net/http"
)

func GetError(_ErrorCode string, _Message string, _Success bool) string {
	result, _ := json.Marshal(struct {
		Code    string `json:"CODE"`
		Message string `json:"MESSAGE"`
		Success bool   `json:"SUCCESS"`
	}{
		Code:    _ErrorCode,
		Message: _Message,
		Success: _Success,
	})
	return string(result)
}

func WriteError(massage string, code int) {
	http.Error(W, massage, code)
}
