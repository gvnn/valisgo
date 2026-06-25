package database

import (
	"testing"
)

func TestNewConnection_SQLiteInMemory(t *testing.T) {
	db, err := NewConnection("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if db == nil {
		t.Fatal("expected db to be non-nil")
	}
}

func TestNewConnection_UnsupportedDriver(t *testing.T) {
	_, err := NewConnection("postgres", "dsn")
	if err == nil {
		t.Fatal("expected error for unsupported driver, got nil")
	}
}
