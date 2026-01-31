// Package base provides base model definitions and common database operations tests
// Author: Done-0
// Created: 2026-01-31
package base

import (
	"encoding/json"
	"testing"
)

func TestJSONMap_ScanNil(t *testing.T) {
	var j JSONMap
	err := j.Scan(nil)
	if err != nil {
		t.Errorf("Scan(nil) error = %v", err)
	}
	if j == nil {
		t.Error("Scan(nil) should initialize empty map")
	}
	if len(j) != 0 {
		t.Errorf("Scan(nil) should create empty map, got %v", j)
	}
}

func TestJSONMap_ScanBytes(t *testing.T) {
	var j JSONMap
	data := []byte(`{"key": "value", "number": 42}`)

	err := j.Scan(data)
	if err != nil {
		t.Errorf("Scan(bytes) error = %v", err)
	}

	if j["key"] != "value" {
		t.Errorf("j[key] = %v, want value", j["key"])
	}
}

func TestJSONMap_ScanInvalidBytes(t *testing.T) {
	var j JSONMap
	data := []byte(`invalid json`)

	err := j.Scan(data)
	if err == nil {
		t.Error("Scan(invalid json) should return error")
	}
}

func TestJSONMap_ScanUnsupportedType(t *testing.T) {
	var j JSONMap
	err := j.Scan(12345)
	if err == nil {
		t.Error("Scan(int) should return error")
	}
}

func TestJSONMap_Value(t *testing.T) {
	j := JSONMap{
		"string": "hello",
		"number": 42,
		"bool":   true,
	}

	value, err := j.Value()
	if err != nil {
		t.Errorf("Value() error = %v", err)
	}

	bytes, ok := value.([]byte)
	if !ok {
		t.Fatal("Value() should return []byte")
	}

	var result map[string]any
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Errorf("Value() returned invalid JSON: %v", err)
	}

	if result["string"] != "hello" {
		t.Errorf("result[string] = %v, want hello", result["string"])
	}
}

func TestJSONMap_RoundTrip(t *testing.T) {
	original := JSONMap{
		"key1": "value1",
		"key2": float64(123), // JSON numbers are float64
		"nested": map[string]any{
			"inner": "data",
		},
	}

	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	var restored JSONMap
	err = restored.Scan(value)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if restored["key1"] != original["key1"] {
		t.Errorf("Round-trip failed for key1: got %v, want %v", restored["key1"], original["key1"])
	}
}

func TestBase_InitialState(t *testing.T) {
	b := Base{}

	if b.ID != 0 {
		t.Errorf("Initial ID should be 0, got %d", b.ID)
	}
	if b.CreatedAt != 0 {
		t.Errorf("Initial CreatedAt should be 0, got %d", b.CreatedAt)
	}
	if b.UpdatedAt != 0 {
		t.Errorf("Initial UpdatedAt should be 0, got %d", b.UpdatedAt)
	}
	if b.Deleted != false {
		t.Error("Initial Deleted should be false")
	}
}

func TestJSONMap_EmptyValue(t *testing.T) {
	j := JSONMap{}

	value, err := j.Value()
	if err != nil {
		t.Errorf("Value() error = %v", err)
	}

	bytes, ok := value.([]byte)
	if !ok {
		t.Fatal("Value() should return []byte")
	}

	if string(bytes) != "{}" {
		t.Errorf("Empty JSONMap should serialize to {}, got %s", string(bytes))
	}
}

func TestJSONMap_ComplexTypes(t *testing.T) {
	j := JSONMap{
		"array":  []any{1, 2, 3},
		"null":   nil,
		"nested": map[string]any{"a": "b"},
	}

	value, err := j.Value()
	if err != nil {
		t.Errorf("Value() with complex types error = %v", err)
	}

	var restored JSONMap
	err = restored.Scan(value)
	if err != nil {
		t.Errorf("Scan() complex types error = %v", err)
	}
}

func BenchmarkJSONMap_Scan(b *testing.B) {
	data := []byte(`{"key1": "value1", "key2": 123, "key3": true}`)

	for i := 0; i < b.N; i++ {
		var j JSONMap
		j.Scan(data)
	}
}

func BenchmarkJSONMap_Value(b *testing.B) {
	j := JSONMap{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	for i := 0; i < b.N; i++ {
		j.Value()
	}
}
