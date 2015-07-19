package gowork

import (
	"reflect"
	"testing"
)

const (
	ERROR_MSG string = "AHH ERROR HAPPENED"
)

func TestNewEventError(t *testing.T) {
	var (
		ofType string
	)

	e := NewEventError(ERROR_MSG)
	ofType = reflect.TypeOf(e).String()

	if ofType != "*gowork.Event" {
		t.Fatalf("Expected type of *Event, received %s instead.", ofType)
	}

	if e.Error != ERROR_MSG {
		t.Fatalf("Expected error message %s, received %s instead.", ERROR_MSG, e.Error)
	}
}

func TestNewEventWork(t *testing.T) {
	var (
		ofType string
	)

	w := NewEventWork(&Work{})
	ofType = reflect.TypeOf(w).String()

	if ofType != "*gowork.Event" {
		t.Fatalf("Expected type of *Event, received %s instead.", ofType)
	}
}

func TestNewEventWorker(t *testing.T) {
	var (
		ofType string
	)

	w := NewEventWorker(&Worker{})
	ofType = reflect.TypeOf(w).String()

	if ofType != "*gowork.Event" {
		t.Fatalf("Expected type of *Event, received %s instead.", ofType)
	}
}
