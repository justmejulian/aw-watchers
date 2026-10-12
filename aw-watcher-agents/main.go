// aw-watcher-agents logs running coding agents to ActivityWatch, via `abtop --json`.
//
// One event per session per status (Executing, Thinking, Waiting, Done), grown
// in place while the status holds. Bucket: aw-watcher-agents_<host>.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"time"

	"aw-watchers/internal/aw"
)

const pollInterval = 15 * time.Second

// a gap longer than this means the Mac slept; don't stretch events over it
const staleAfter = 3 * pollInterval

type abtopSession struct {
	AgentCLI    string `json:"agent_cli"`
	SessionID   string `json:"session_id"`
	Status      string `json:"status"`
	ProjectName string `json:"project_name"`
	Cwd         string `json:"cwd"`
	GitBranch   string `json:"git_branch"`
	Model       string `json:"model"`
	Summary     string `json:"summary"`
}

type session struct {
	id    *int64
	start time.Time
	last  time.Time
	data  map[string]any
}

type inserter func(aw.Event) (aw.Event, error)

type watcher struct {
	insert   inserter
	sessions map[string]*session
}

func sessionData(s abtopSession) map[string]any {
	return map[string]any{
		"agent":      s.AgentCLI,
		"status":     s.Status,
		"project":    s.ProjectName,
		"cwd":        s.Cwd,
		"branch":     s.GitBranch,
		"model":      s.Model,
		"title":      s.Summary,
		"session_id": s.SessionID,
	}
}

// write inserts the event, or replaces it in place once AW has given it an id.
func (w *watcher) write(s *session) {
	inserted, err := w.insert(aw.Event{
		ID:        s.id,
		Timestamp: s.start,
		Duration:  s.last.Sub(s.start).Seconds(),
		Data:      s.data,
	})
	if err != nil {
		log.Printf("write event: %v", err)
		return
	}
	s.id = inserted.ID
}

func (w *watcher) update(sessions []abtopSession, now time.Time) {
	seen := map[string]bool{}
	for _, as := range sessions {
		seen[as.SessionID] = true
		data := sessionData(as)
		s := w.sessions[as.SessionID]
		stale := s != nil && now.Sub(s.last) > staleAfter
		if s != nil && !stale && s.data["status"] == data["status"] {
			s.last = now
			s.data = data
		} else {
			if s != nil && !stale {
				s.last = now
				w.write(s)
			}
			s = &session{start: now, last: now, data: data}
			w.sessions[as.SessionID] = s
		}
		w.write(s)
	}
	for id := range w.sessions {
		if !seen[id] {
			delete(w.sessions, id)
		}
	}
}

func snapshot(abtop string) ([]abtopSession, error) {
	out, err := exec.Command(abtop, "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("abtop: %w", err)
	}
	var snap struct {
		Sessions []abtopSession `json:"sessions"`
	}
	if err := json.Unmarshal(out, &snap); err != nil {
		return nil, fmt.Errorf("abtop output: %w", err)
	}
	return snap.Sessions, nil
}

func main() {
	testing := flag.Bool("testing", false, "use the aw-server testing instance (port 5666)")
	// aw-qt starts watchers without the shell PATH, so default to the full path
	abtop := flag.String("abtop", "/opt/homebrew/bin/abtop", "path to the abtop binary")
	flag.Parse()

	client := aw.New(*testing)
	var bucket string
	for {
		host, err := client.Hostname()
		if err == nil {
			bucket = "aw-watcher-agents_" + host
			err = client.CreateBucket(bucket, "coding-agent", "aw-watcher-agents", host)
		}
		if err == nil {
			break
		}
		log.Printf("waiting for aw-server: %v", err)
		time.Sleep(5 * time.Second)
	}

	w := &watcher{
		insert:   func(e aw.Event) (aw.Event, error) { return client.InsertEvent(bucket, e) },
		sessions: map[string]*session{},
	}
	for {
		sessions, err := snapshot(*abtop)
		if err != nil {
			log.Print(err)
		} else {
			w.update(sessions, time.Now())
		}
		time.Sleep(pollInterval)
	}
}
