// Package rate provides rate limiting utilities tests
// Author: Done-0
// Created: 2026-01-31
package rate

import (
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantLimit  rate.Limit
		wantBurst  int
		wantErr    bool
		checkLimit bool // Whether to check exact limit value
	}{
		{
			name:       "60 per minute",
			input:      "60/min",
			wantBurst:  60,
			wantErr:    false,
			checkLimit: false,
		},
		{
			name:       "1 per second",
			input:      "1/s",
			wantLimit:  rate.Every(time.Second),
			wantBurst:  1,
			wantErr:    false,
			checkLimit: true,
		},
		{
			name:       "100 per second",
			input:      "100/s",
			wantBurst:  100,
			wantErr:    false,
			checkLimit: false,
		},
		{
			name:       "10 per minute",
			input:      "10/minute",
			wantBurst:  10,
			wantErr:    false,
			checkLimit: false,
		},
		{
			name:       "5 per hour",
			input:      "5/h",
			wantBurst:  5,
			wantErr:    false,
			checkLimit: false,
		},
		{
			name:       "1 per hour",
			input:      "1/hour",
			wantBurst:  1,
			wantErr:    false,
			checkLimit: false,
		},
		{
			name:      "invalid format no slash",
			input:     "100",
			wantErr:   true,
		},
		{
			name:      "invalid format multiple slashes",
			input:     "100/s/extra",
			wantErr:   true,
		},
		{
			name:      "invalid requests",
			input:     "abc/s",
			wantErr:   true,
		},
		{
			name:      "invalid duration",
			input:     "100/invalid",
			wantErr:   true,
		},
		{
			name:      "empty string",
			input:     "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotBurst, err := ParseLimit(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if gotBurst != tt.wantBurst {
				t.Errorf("ParseLimit() burst = %d, want %d", gotBurst, tt.wantBurst)
			}
			if tt.checkLimit && gotLimit != tt.wantLimit {
				t.Errorf("ParseLimit() limit = %v, want %v", gotLimit, tt.wantLimit)
			}
			// Verify limit is positive
			if gotLimit <= 0 {
				t.Errorf("ParseLimit() returned non-positive limit: %v", gotLimit)
			}
		})
	}
}

func TestParseLimit_DurationVariants(t *testing.T) {
	// Test all duration format variants
	secondVariants := []string{"s", "sec", "second"}
	for _, v := range secondVariants {
		_, burst, err := ParseLimit("10/" + v)
		if err != nil {
			t.Errorf("ParseLimit(10/%s) error = %v", v, err)
		}
		if burst != 10 {
			t.Errorf("ParseLimit(10/%s) burst = %d, want 10", v, burst)
		}
	}

	minuteVariants := []string{"m", "min", "minute"}
	for _, v := range minuteVariants {
		_, burst, err := ParseLimit("60/" + v)
		if err != nil {
			t.Errorf("ParseLimit(60/%s) error = %v", v, err)
		}
		if burst != 60 {
			t.Errorf("ParseLimit(60/%s) burst = %d, want 60", v, burst)
		}
	}

	hourVariants := []string{"h", "hour"}
	for _, v := range hourVariants {
		_, burst, err := ParseLimit("100/" + v)
		if err != nil {
			t.Errorf("ParseLimit(100/%s) error = %v", v, err)
		}
		if burst != 100 {
			t.Errorf("ParseLimit(100/%s) burst = %d, want 100", v, burst)
		}
	}
}

func TestParseLimit_CustomDuration(t *testing.T) {
	// Test with Go duration format
	limit, burst, err := ParseLimit("10/30s")
	if err != nil {
		t.Errorf("ParseLimit(10/30s) error = %v", err)
	}
	if burst != 10 {
		t.Errorf("ParseLimit(10/30s) burst = %d, want 10", burst)
	}
	if limit <= 0 {
		t.Error("ParseLimit(10/30s) should return positive limit")
	}
}

func TestParseLimit_RateLimiterIntegration(t *testing.T) {
	// Test that parsed limit works with rate.Limiter
	limit, burst, err := ParseLimit("2/s")
	if err != nil {
		t.Fatalf("ParseLimit() error = %v", err)
	}

	limiter := rate.NewLimiter(limit, burst)

	// Should allow initial burst
	for i := 0; i < burst; i++ {
		if !limiter.Allow() {
			t.Errorf("Limiter should allow request %d within burst", i)
		}
	}
}

func BenchmarkParseLimit(b *testing.B) {
	inputs := []string{"60/min", "100/s", "1000/h"}
	for i := 0; i < b.N; i++ {
		_, _, err := ParseLimit(inputs[i%len(inputs)])
		if err != nil {
			b.Fatalf("ParseLimit() error = %v", err)
		}
	}
}
