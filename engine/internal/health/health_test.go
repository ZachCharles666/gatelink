package health

import (
	"testing"
)

func TestHTTPStatusToEvent(t *testing.T) {
	cases := []struct {
		code  int
		event EventType
	}{
		{200, EventSuccess},
		{201, EventSuccess},
		{401, EventUnauthorized},
		{403, EventUnauthorized},
		{429, EventRateLimited},
		{400, EventClient4xx},
		{404, EventClient4xx},
		{500, EventServer5xx},
		{503, EventServer5xx},
	}
	for _, c := range cases {
		got := HTTPStatusToEvent(c.code)
		if got != c.event {
			t.Errorf("HTTPStatusToEvent(%d) = %q, want %q", c.code, got, c.event)
		}
	}
}

func TestClamp(t *testing.T) {
	cases := []struct {
		v, min, max, want int
	}{
		{50, 0, 100, 50},
		{-10, 0, 100, 0},
		{150, 0, 100, 100},
		{100, 0, 100, 100},
		{0, 0, 100, 0},
	}
	for _, c := range cases {
		got := clamp(c.v, c.min, c.max)
		if got != c.want {
			t.Errorf("clamp(%d, %d, %d) = %d, want %d", c.v, c.min, c.max, got, c.want)
		}
	}
}

func TestMarshalDetail(t *testing.T) {
	got := marshalDetail(nil)
	if got != "null" {
		t.Errorf("nil detail should marshal to 'null', got %q", got)
	}

	got = marshalDetail(map[string]interface{}{"status_code": 200})
	if len(got) == 0 {
		t.Error("non-nil detail should produce non-empty JSON")
	}
}

func TestIsFailure(t *testing.T) {
	if isFailure(200) {
		t.Error("200 should not be failure")
	}
	if !isFailure(400) {
		t.Error("400 should be failure")
	}
	if !isFailure(500) {
		t.Error("500 should be failure")
	}
}
