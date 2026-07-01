package registries

import "net/http"

type fallbackResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *fallbackResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.status = statusCode
	if statusCode != http.StatusNotFound {
		w.ResponseWriter.WriteHeader(statusCode)
		w.wroteHeader = true
	}
}

func (w *fallbackResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.status == http.StatusNotFound {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
