// Package validator provides parameter validation utilities tests
// Author: Done-0
// Created: 2026-01-31
package validator

import (
	"testing"
)

// TestStruct for validation testing
type TestStruct struct {
	Name     string `validate:"required,min=2,max=50"`
	Email    string `validate:"required,email"`
	Age      int    `validate:"gte=0,lte=150"`
	Password string `validate:"required,min=6"`
}

func TestValidate_ValidData(t *testing.T) {
	data := TestStruct{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      25,
		Password: "secret123",
	}

	errors := Validate(data)
	if len(errors) != 0 {
		t.Errorf("Validate() returned errors for valid data: %v", errors)
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	data := TestStruct{
		Name:     "",
		Email:    "",
		Age:      0,
		Password: "",
	}

	errors := Validate(data)
	if len(errors) == 0 {
		t.Error("Validate() should return errors for missing required fields")
	}

	// Check that we got errors for required fields
	requiredFields := map[string]bool{
		"Name":     false,
		"Email":    false,
		"Password": false,
	}

	for _, err := range errors {
		if _, ok := requiredFields[err.Field]; ok {
			requiredFields[err.Field] = true
		}
	}

	for field, found := range requiredFields {
		if !found {
			t.Errorf("Expected error for required field %s", field)
		}
	}
}

func TestValidate_InvalidEmail(t *testing.T) {
	data := TestStruct{
		Name:     "John",
		Email:    "invalid-email",
		Age:      25,
		Password: "secret123",
	}

	errors := Validate(data)
	if len(errors) == 0 {
		t.Error("Validate() should return error for invalid email")
	}

	found := false
	for _, err := range errors {
		if err.Field == "Email" && err.Tag == "email" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected email validation error")
	}
}

func TestValidate_MinLength(t *testing.T) {
	data := TestStruct{
		Name:     "A", // Too short, min=2
		Email:    "test@test.com",
		Age:      25,
		Password: "123", // Too short, min=6
	}

	errors := Validate(data)
	if len(errors) < 2 {
		t.Errorf("Validate() should return at least 2 errors, got %d", len(errors))
	}
}

func TestValidate_RangeValidation(t *testing.T) {
	tests := []struct {
		name      string
		age       int
		expectErr bool
	}{
		{"valid age", 25, false},
		{"zero age", 0, false},
		{"max age", 150, false},
		{"negative age", -1, true},
		{"over max age", 151, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := TestStruct{
				Name:     "Test",
				Email:    "test@test.com",
				Age:      tt.age,
				Password: "secret123",
			}

			errors := Validate(data)
			hasAgeError := false
			for _, err := range errors {
				if err.Field == "Age" {
					hasAgeError = true
					break
				}
			}

			if hasAgeError != tt.expectErr {
				t.Errorf("Age=%d, expectErr=%v, but got hasAgeError=%v", tt.age, tt.expectErr, hasAgeError)
			}
		})
	}
}

func TestValidErrRes_Fields(t *testing.T) {
	data := TestStruct{
		Name:     "",
		Email:    "test@test.com",
		Age:      25,
		Password: "secret123",
	}

	errors := Validate(data)
	if len(errors) == 0 {
		t.Fatal("Expected at least one error")
	}

	err := errors[0]
	if !err.Error {
		t.Error("ValidErrRes.Error should be true")
	}
	if err.Field == "" {
		t.Error("ValidErrRes.Field should not be empty")
	}
	if err.Tag == "" {
		t.Error("ValidErrRes.Tag should not be empty")
	}
}

func TestValidate_Pointer(t *testing.T) {
	data := &TestStruct{
		Name:     "Test",
		Email:    "test@test.com",
		Age:      25,
		Password: "secret123",
	}

	errors := Validate(data)
	if len(errors) != 0 {
		t.Errorf("Validate() with pointer returned errors: %v", errors)
	}
}

// NestedStruct for nested validation testing
type NestedStruct struct {
	User    TestStruct `validate:"required"`
	Comment string     `validate:"required,max=1000"`
}

func TestValidate_NestedStruct(t *testing.T) {
	data := NestedStruct{
		User: TestStruct{
			Name:     "Test",
			Email:    "test@test.com",
			Age:      25,
			Password: "secret123",
		},
		Comment: "This is a comment",
	}

	errors := Validate(data)
	if len(errors) != 0 {
		t.Errorf("Validate() returned errors for valid nested struct: %v", errors)
	}
}

func TestNewValidator(t *testing.T) {
	if NewValidator == nil {
		t.Error("NewValidator should not be nil")
	}
}

func BenchmarkValidate(b *testing.B) {
	data := TestStruct{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      25,
		Password: "secret123",
	}

	for i := 0; i < b.N; i++ {
		Validate(data)
	}
}
