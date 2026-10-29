package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutCancelsRequestContext(t *testing.T) {
	finished := make(chan struct{})
	handler := timeoutHandler(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		close(finished)
	}), time.Millisecond)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("request context was not canceled at its deadline")
	}
}
