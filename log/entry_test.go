package log_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/TouchBistro/goutils/log"
)

func TestCopy(t *testing.T) {
	e1 := &log.Entry{
		Fields: log.Fields{
			"foo": "bar",
			"baz": "qux",
		},
		Time: time.Now(),
	}
	e2 := e1.Copy()

	if !reflect.DeepEqual(e1, e2) {
		t.Fatalf("want entries to be equal, got\n\ne1: %+v\ne2: %+v", e1, e2)
	}

	// Should not affect e1 since the map should have been copied
	e2.Fields["first"] = "yes"
	if reflect.DeepEqual(e1, e2) {
		t.Fatal("want entries to not be equal, but are")
	}
}

func TestWithFields(t *testing.T) {
	e1 := &log.Entry{}
	if len(e1.Fields) != 0 {
		t.Fatalf("want entry fields to be empty, got %+v", e1.Fields)
	}

	e2 := e1.WithFields(log.Fields{"foo": "bar"}).(*log.Entry)
	e3 := e2.WithFields(log.Fields{"baz": "qux"}).(*log.Entry)

	wantFields := log.Fields{"foo": "bar", "baz": "qux"}
	if !reflect.DeepEqual(e3.Fields, wantFields) {
		t.Fatalf("got entry fields: %+v\nwant: %+v", e3.Fields, wantFields)
	}

	// Make sure e2 was not mutated
	if reflect.DeepEqual(e2.Fields, e3.Fields) {
		t.Fatal("want entry fields to not be equal, but are")
	}
}
