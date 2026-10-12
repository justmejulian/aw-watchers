package main

import (
	"testing"
	"time"

	"aw-watchers/internal/aw"
)

// fakeAW records the latest version of each event, keyed by id.
type fakeAW struct {
	nextID int64
	events map[int64]aw.Event
}

func (f *fakeAW) insert(e aw.Event) (aw.Event, error) {
	if e.ID == nil {
		f.nextID++
		id := f.nextID
		e.ID = &id
	}
	f.events[*e.ID] = e
	return e, nil
}

func newWatcher() (*watcher, *fakeAW) {
	f := &fakeAW{events: map[int64]aw.Event{}}
	return &watcher{insert: f.insert, sessions: map[string]*session{}}, f
}

func sess(status string) []abtopSession {
	return []abtopSession{{AgentCLI: "claude", SessionID: "s1", Status: status}}
}

func TestSameStatusGrowsOneEvent(t *testing.T) {
	w, f := newWatcher()
	t0 := time.Unix(1000, 0)
	w.update(sess("Executing"), t0)
	w.update(sess("Executing"), t0.Add(pollInterval))
	w.update(sess("Executing"), t0.Add(2*pollInterval))

	if len(f.events) != 1 {
		t.Fatalf("got %d events, want 1", len(f.events))
	}
	if d := f.events[1].Duration; d != 30 {
		t.Errorf("duration = %v, want 30", d)
	}
}

func TestStatusChangeStartsNewEvent(t *testing.T) {
	w, f := newWatcher()
	t0 := time.Unix(1000, 0)
	w.update(sess("Executing"), t0)
	w.update(sess("Waiting"), t0.Add(pollInterval))

	if len(f.events) != 2 {
		t.Fatalf("got %d events, want 2", len(f.events))
	}
	if d := f.events[1].Duration; d != 15 {
		t.Errorf("first event duration = %v, want 15 (closed at status change)", d)
	}
	if s := f.events[2].Data["status"]; s != "Waiting" {
		t.Errorf("second event status = %v, want Waiting", s)
	}
}

func TestSleepGapStartsNewEvent(t *testing.T) {
	w, f := newWatcher()
	t0 := time.Unix(1000, 0)
	w.update(sess("Executing"), t0)
	w.update(sess("Executing"), t0.Add(pollInterval))
	w.update(sess("Executing"), t0.Add(time.Hour))

	if len(f.events) != 2 {
		t.Fatalf("got %d events, want 2", len(f.events))
	}
	if d := f.events[1].Duration; d != 15 {
		t.Errorf("event before sleep duration = %v, want 15", d)
	}
}

func TestEndedSessionIsForgotten(t *testing.T) {
	w, _ := newWatcher()
	w.update(sess("Executing"), time.Unix(1000, 0))
	w.update(nil, time.Unix(1015, 0))

	if len(w.sessions) != 0 {
		t.Errorf("got %d tracked sessions, want 0", len(w.sessions))
	}
}
