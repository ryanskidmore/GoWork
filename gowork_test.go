package gowork

import (
	"reflect"
	"testing"
)

const (
	ERROR_MSG          string = "AHH ERROR HAPPENED"
	EVENT_ID           string = "123abc"
	SECRET_STR_INVALID string = "GoWork"
	SECRET_STR_VALID   string = "GoWorkGoWorkGoWorkGoWorkGoWork12"
)

func TestNewEventError(t *testing.T) {
	var (
		ofType string
	)

	t.Parallel()

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

	t.Parallel()

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

	t.Parallel()

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

	t.Parallel()

	if _, err = NewServer(SECRET_STR_VALID); err != nil {
		t.Fatal(err)
	}
}

func TestNewServerInvalidSecretSize(t *testing.T) {
	var (
		err error
	)

	t.Parallel()

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

	t.Parallel()
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

	t.Parallel()
	MustNewServer(SECRET_STR_INVALID)
}

func TestNewHandler(t *testing.T) {
	var (
		err error
		ws  *WorkServer = MustNewServer(SECRET_STR_VALID)
	)

	t.Parallel()

	if err = ws.NewHandler(EVENT_ID, func(*Event, map[string]interface{}) {}); err != nil {
		t.Fatal(err)
	}

	if _, ok := ws.Handlers[EVENT_ID]; !ok {
		t.Fatalf("Expected event_id %s to be set as a handler, it was not", EVENT_ID)
	}
}

func TestNewHandlerAlreadyExists(t *testing.T) {
	var (
		err error
		ws  *WorkServer = MustNewServer(SECRET_STR_VALID)
	)

	t.Parallel()

	if err = ws.NewHandler(EVENT_ID, func(*Event, map[string]interface{}) {}); err != nil {
		t.Fatal(err)
	}

	if err = ws.NewHandler(EVENT_ID, func(*Event, map[string]interface{}) {}); err == nil {
		t.Fatalf("Expected NewHandler to throw a 'Handler already exists' error.  Event ID %s was passed in.", EVENT_ID)
	}
}

func TestAddParams(t *testing.T) {
	var (
		ws     *WorkServer            = MustNewServer(SECRET_STR_VALID)
		params map[string]interface{} = map[string]interface{}{
			"key":  "value",
			"key2": 12,
		}
	)

	t.Parallel()

	ws = ws.AddParams(params)

	if ws.HandlerParams["key"].(string) != "value" {
		t.Fatal("AddParams failed to set HandlerParams properly.")
	}

	if ws.HandlerParams["key2"].(int) != 12 {
		t.Fatal("AddParams failed to set HandlerParams properly.")
	}
}

func TestQueueSize(t *testing.T) {
	var (
		ws *WorkServer = MustNewServer(SECRET_STR_VALID)
	)

	if ws.QueueSize() != ws.Queue.Size() {
		t.Fatalf("Expected QueueSize to return %d, it returned %d instead.", ws.Queue.Size(), ws.QueueSize())
	}
}
