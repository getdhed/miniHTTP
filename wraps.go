package main

import (
	"encoding/json"
	"net/http"
)

func writeError(w http.ResponseWriter, status int, code string, message string) error {

	resp := errorResponse{
		Error:   code,
		Message: message,
	}

	return writeJSON(w, status, resp)
}
func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(rw.status)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}

	n, err := rw.ResponseWriter.Write(data)

	rw.bytes += n

	return n, err
}

func writeJSON(w http.ResponseWriter, status int, data any) error {
	resp, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(resp); err != nil {
		return err
	}

	return nil
}
