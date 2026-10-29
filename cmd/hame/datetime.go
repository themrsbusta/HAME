package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const datetimeMarker = "HAME_RUNTIME_DATETIME: "

type DateTimeResult struct {
	Datetime         string `json:"datetime"`
	Date             string `json:"date"`
	Time             string `json:"time"`
	Timezone         string `json:"timezone"`
	UTCOffsetSeconds int    `json:"utc_offset_seconds"`
	Unix             int64  `json:"unix"`
	Source           string `json:"source"`
}

func getDatetime(now time.Time) DateTimeResult {
	zone, offset := now.Zone()
	return DateTimeResult{now.Format(time.RFC3339), now.Format("2006-01-02"), now.Format("15:04:05"), zone, offset, now.Unix(), "system_clock"}
}
func withDatetime(messages []Message, now time.Time) []Message {
	data, _ := json.Marshal(getDatetime(now))
	reading := datetimeMarker + string(data) + "\nThis is the current host system date/time, sampled for this request. Use it for current date/time questions; older dates in conversation history describe past turns. Do not claim to have called a tool: HAME supplied this reading automatically."
	result := []Message{}
	for _, m := range messages {
		if m.Role == "system" {
			if i := strings.Index(m.Content, datetimeMarker); i >= 0 {
				m.Content = strings.TrimSpace(m.Content[:i])
				if m.Content == "" {
					continue
				}
			}
		}
		result = append(result, m)
	}
	if len(result) > 0 && result[0].Role == "system" {
		result[0].Content += "\n" + reading
	} else {
		result = append([]Message{{Role: "system", Content: reading}}, result...)
	}
	return result
}
func datetimeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "GET required", 405)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(getDatetime(time.Now()))
}
