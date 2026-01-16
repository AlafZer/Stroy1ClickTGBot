package telegram

import (
	"encoding/json"
	"io"
	"net/http"
)

func extractToken(w http.ResponseWriter, r *http.Request, data any) error {
	var maxBytes int64 = 1048576

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	dec := json.NewDecoder(r.Body)

	err := dec.Decode(data)
	if err != nil {
		return err
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return err
	}

	return nil
}
