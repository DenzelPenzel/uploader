package stats_test

import (
	"encoding/json"
	"github.com/denisschmidt/uploader/internal/stats"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

var testHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("test"))
})

func TestSingleRequest(t *testing.T) {
	s := stats.NewStatistic()

	rec := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	s.WrapHandler(testHandler).ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	require.True(t, reflect.DeepEqual(map[string]int{"200": 1}, s.ResponseCounts))
	require.Equal(t, 1, s.TotalRespCounts["200"])

}

func TestGetStats(t *testing.T) {
	s := stats.NewStatistic()

	var fn = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := s.GatherData()
		// encode into JSON
		b, _ := json.Marshal(res)

		w.Write(b)
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
	})

	rec := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	s.WrapHandler(testHandler).ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	rec = httptest.NewRecorder()
	s.WrapHandler(fn).ServeHTTP(rec, req)

	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	data := make(map[string]interface{})
	// decode JSON
	err = json.Unmarshal(rec.Body.Bytes(), &data)
	require.NoError(t, err)

	require.Equal(t, float64(1), data["total_count"].(float64))
}

func TestRace(t *testing.T) {
	stat := stats.NewStatistic()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	wrappedHandler := stat.WrapHandler(handler)

	ch1 := make(chan bool)
	ch2 := make(chan bool)

	go func() {
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()

		for true {
			select {
			case _ = <-ch1:
				return
			default:
				wrappedHandler.ServeHTTP(rr, req)
			}
		}
	}()

	go func() {
		for true {
			select {
			case _ = <-ch2:
				return
			default:
				data := stat.GatherData()
				_ = data.TotalStatusCodeCount["200"]
			}
		}
	}()

	time.Sleep(time.Second)

	ch1 <- true
	ch2 <- true
}

func TestIgnoreHijackedConnection(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Hijacker).Hijack()
	})

	stat := stats.NewStatistic()

	rec := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	wrappedHandler := stat.WrapHandler(handler)
	wrappedHandler.ServeHTTP(rec, req)

	require.True(t, reflect.DeepEqual(map[string]int{}, stat.ResponseCounts))
}
