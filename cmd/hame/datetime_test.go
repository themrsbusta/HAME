package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCurrentDatetimeReplacesOldReading(t *testing.T) {
	zone := time.FixedZone("test-zone", -3*3600)
	now := time.Date(2031, 2, 3, 4, 5, 6, 0, zone)
	original := []Message{{Role: "system", Content: "memory"}, {Role: "user", Content: "hello"}, {Role: "assistant", Content: "reply"}}
	first := withDatetime(original, now)
	next := withDatetime(first, now.Add(24*time.Hour))
	count := 0
	for _, m := range next {
		if strings.Contains(m.Content, datetimeMarker) {
			count++
			if !strings.Contains(m.Content, "2031-02-04T04:05:06-03:00") {
				t.Fatal(m)
			}
		}
	}
	if count != 1 || len(next) != 3 || len(original) != 3 {
		t.Fatal(next)
	}
	d := getDatetime(now)
	if d.UTCOffsetSeconds != -10800 || d.Date != "2031-02-03" || d.Source != "system_clock" {
		t.Fatal(d)
	}
}
func TestDatetimeHTTP(t *testing.T) {
	w := httptest.NewRecorder()
	before := time.Now().Unix()
	datetimeHTTP(w, httptest.NewRequest("GET", "/hame/get_datetime", nil))
	var got DateTimeResult
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || got.Unix < before || got.Unix > time.Now().Unix() || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(got, w.Code)
	}
	w = httptest.NewRecorder()
	datetimeHTTP(w, httptest.NewRequest("POST", "/hame/get_datetime", nil))
	if w.Code != 405 {
		t.Fatal(w.Code)
	}
}
