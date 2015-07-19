package gowork

import (
	"reflect"
	"testing"
)

const (
	ERROR_MSG          string = "AHH ERROR HAPPENED"
	SECRET_STR_INVALID string = "GoWork"
	SECRET_STR_VALID   string = "GoWorkGoWorkGoWorkGoWorkGoWork12"
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

func TestNewServer(t *testing.T) {
	var (
		err error
	)

	if _, err = NewServer(SECRET_STR_VALID); err != nil {
		t.Fatal(err)
	}
}

func TestNewServerInvalidSecretSize(t *testing.T) {
	var (
		err error
	)

	if _, err = NewServer(SECRET_STR_INVALID); err == nil {
		t.Fatalf("Expected NewServer to throw a secret length error.  Secret %s was passed in.", SECRET_STR_INVALID)
	}
}

func TestMustNewServer(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("TestMustNewServer paniced when it shouldn't have. Recovered %v", r)
		}
	}()

	MustNewServer(SECRET_STR_VALID)
}

func TestMustNewServerPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			return
		} else {
			t.Fatalf("TestMustNewServer did not panic when it should have. Secret %s was passed in.", SECRET_STR_INVALID)
		}
	}()

	MustNewServer(SECRET_STR_INVALID)
}
