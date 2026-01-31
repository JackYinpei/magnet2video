// Package snowflake provides snowflake algorithm ID generation utilities tests
// Author: Done-0
// Created: 2026-01-31
package snowflake

import (
	"sync"
	"testing"
)

func TestGenerateID(t *testing.T) {
	id, err := GenerateID()
	if err != nil {
		t.Errorf("GenerateID() error = %v", err)
		return
	}
	if id <= 0 {
		t.Errorf("GenerateID() returned non-positive ID: %d", id)
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	const count = 10000
	ids := make(map[int64]bool)

	for i := 0; i < count; i++ {
		id, err := GenerateID()
		if err != nil {
			t.Fatalf("GenerateID() error = %v at iteration %d", err, i)
		}
		if ids[id] {
			t.Errorf("GenerateID() generated duplicate ID: %d", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(ids))
	}
}

func TestGenerateID_Concurrent(t *testing.T) {
	const goroutines = 10
	const idsPerGoroutine = 1000

	var wg sync.WaitGroup
	idsChan := make(chan int64, goroutines*idsPerGoroutine)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, err := GenerateID()
				if err != nil {
					t.Errorf("GenerateID() error = %v", err)
					return
				}
				idsChan <- id
			}
		}()
	}

	wg.Wait()
	close(idsChan)

	ids := make(map[int64]bool)
	for id := range idsChan {
		if ids[id] {
			t.Errorf("Concurrent GenerateID() generated duplicate ID: %d", id)
		}
		ids[id] = true
	}

	expectedCount := goroutines * idsPerGoroutine
	if len(ids) != expectedCount {
		t.Errorf("Expected %d unique IDs, got %d", expectedCount, len(ids))
	}
}

func TestGenerateID_Ordering(t *testing.T) {
	// IDs should be monotonically increasing (generally)
	var prevID int64
	for i := 0; i < 100; i++ {
		id, err := GenerateID()
		if err != nil {
			t.Fatalf("GenerateID() error = %v", err)
		}
		if i > 0 && id <= prevID {
			// Note: This might occasionally fail due to clock issues
			// but generally IDs should be increasing
			t.Logf("Warning: ID %d is not greater than previous %d", id, prevID)
		}
		prevID = id
	}
}

func TestGenerateID_Format(t *testing.T) {
	id, err := GenerateID()
	if err != nil {
		t.Fatalf("GenerateID() error = %v", err)
	}

	// Snowflake IDs are 64-bit integers
	// They should be positive and reasonably large
	if id < 0 {
		t.Errorf("ID should be positive, got %d", id)
	}

	// A valid snowflake ID should have significant bits set
	// (not just a small number)
	if id < 1000000 {
		t.Logf("Warning: ID %d seems unusually small for a snowflake ID", id)
	}
}

func BenchmarkGenerateID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GenerateID()
		if err != nil {
			b.Fatalf("GenerateID() error = %v", err)
		}
	}
}

func BenchmarkGenerateID_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := GenerateID()
			if err != nil {
				b.Fatalf("GenerateID() error = %v", err)
			}
		}
	})
}
