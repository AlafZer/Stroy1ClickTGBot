package telegram

import (
	"encoding/json"
	"io"
	"net/http"
)

type jsonResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

func (tgWrkr *TGWorker) readJSON(w http.ResponseWriter, r *http.Request, data any) error {
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

func (tgWrkr *TGWorker) writeJSON(w http.ResponseWriter, status int, data any, headers ...http.Header) error {
	out, err := json.Marshal(data)

	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(out)

	if err != nil {
		return err
	}

	return nil
}

func (tgWrkr *TGWorker) errorJSON(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload jsonResponse

	payload.Error = true
	payload.Message = err.Error()

	return tgWrkr.writeJSON(w, statusCode, payload)
}
