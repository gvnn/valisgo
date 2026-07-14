package registry

import "net/http"

// HandleInternalError checks if the error is not nil, writes a 500 response, and returns true.
// It simplifies repetitive error checking in protocol handlers.
func HandleInternalError(w http.ResponseWriter, err error) bool {
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return true
	}
	return false
}
