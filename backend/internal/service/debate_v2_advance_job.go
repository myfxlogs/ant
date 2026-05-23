package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// debateV2JobHub tracks in-process LLM jobs (advance kickoff/code-gen, or chat replies).
// Events are JSON lines pushed to SSE subscribers (see server SSE handler).
type debateV2JobHub struct {
	mu   sync.Mutex
	jobs map[string]*debateV2Job
}

type debateV2Job struct {
	userID    uuid.UUID
	sessionID uuid.UUID
	subsMu    sync.Mutex
	subs      []chan string
	stateMu   sync.Mutex
	phase     string
	message   string
	// streamAcc holds all chunk deltas so SSE clients that subscribe after the
	// goroutine started still receive a catch-up chunk (Unary returns before EventSource opens).
	streamMu  sync.Mutex
	streamAcc strings.Builder
	// Chat-only: set by putPreparedChatJob; RunDebateV2ChatJob reads then clears eligibility via phase transition.
	chatPrepared bool
	chatLocale   string
	chatStepKey  string
}

func newDebateV2JobHub() *debateV2JobHub {
	return &debateV2JobHub{jobs: make(map[string]*debateV2Job)}
}

// startKeepalive emits periodic SSE lines while the upstream LLM is silent.
// Many CDNs and reverse proxies (notably Cloudflare ~100s) close idle streams;
// EventSource ignores unknown events; the client only merges "chunk" content.
func (h *debateV2JobHub) startKeepalive(jobID string) (stop func()) {
	if h == nil || jobID == "" {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		t := time.NewTicker(12 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				h.emit(jobID, `{"event":"ping"}`)
			}
		}
	}()
	return cancel
}

func (h *debateV2JobHub) put(jobID string, userID, sessionID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.jobs[jobID] = &debateV2Job{userID: userID, sessionID: sessionID, phase: "queued"}
}

// putPreparedChatJob registers a chat job in queued state with locale/stepKey for RunDebateV2ChatJob.
func (h *debateV2JobHub) putPreparedChatJob(jobID string, userID, sessionID uuid.UUID, locale, stepKey string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.jobs[jobID] = &debateV2Job{
		userID:       userID,
		sessionID:    sessionID,
		phase:        "queued",
		chatPrepared: true,
		chatLocale:   locale,
		chatStepKey:  stepKey,
	}
}

// tryConsumeQueuedChat transitions a prepared chat job from queued to running and returns
// session id plus metadata for the worker. false if missing, wrong user, not prepared, or not queued.
func (h *debateV2JobHub) tryConsumeQueuedChat(jobID string, userID uuid.UUID) (sessionID uuid.UUID, locale, stepKey string, ok bool) {
	j := h.job(jobID)
	if j == nil {
		return uuid.Nil, "", "", false
	}
	j.stateMu.Lock()
	defer j.stateMu.Unlock()
	if j.userID != userID || !j.chatPrepared || j.phase != "queued" {
		return uuid.Nil, "", "", false
	}
	j.phase = "running"
	return j.sessionID, j.chatLocale, j.chatStepKey, true
}

func (h *debateV2JobHub) job(jobID string) *debateV2Job {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.jobs[jobID]
}

func (h *debateV2JobHub) emit(jobID, line string) {
	j := h.job(jobID)
	if j == nil {
		return
	}
	j.subsMu.Lock()
	defer j.subsMu.Unlock()
	for _, ch := range j.subs {
		select {
		case ch <- line:
		default:
		}
	}
}

// emitChunk records delta for late subscribers and broadcasts one chunk line.
func (h *debateV2JobHub) emitChunk(jobID, delta string) {
	if delta == "" {
		return
	}
	j := h.job(jobID)
	if j == nil {
		return
	}
	j.streamMu.Lock()
	_, _ = j.streamAcc.WriteString(delta)
	j.streamMu.Unlock()
	line, err := json.Marshal(map[string]string{"event": "chunk", "content": delta})
	if err != nil {
		return
	}
	h.emit(jobID, string(line))
}

func (h *debateV2JobHub) setPhase(jobID, phase, msg string) {
	j := h.job(jobID)
	if j == nil {
		return
	}
	j.stateMu.Lock()
	j.phase = phase
	j.message = msg
	j.stateMu.Unlock()
}

func (h *debateV2JobHub) get(jobID string, userID uuid.UUID) (phase, msg, sessionID string, err error) {
	j := h.job(jobID)
	if j == nil {
		return "", "", "", errors.New("job not found")
	}
	if j.userID != userID {
		return "", "", "", errors.New("forbidden")
	}
	j.stateMu.Lock()
	phase, msg = j.phase, j.message
	j.stateMu.Unlock()
	return phase, msg, j.sessionID.String(), nil
}

func (h *debateV2JobHub) subscribe(jobID string, userID uuid.UUID) (<-chan string, func(), error) {
	j := h.job(jobID)
	if j == nil {
		return nil, nil, errors.New("job not found")
	}
	if j.userID != userID {
		return nil, nil, errors.New("forbidden")
	}
	ch := make(chan string, 4096)
	j.subsMu.Lock()
	j.subs = append(j.subs, ch)
	j.subsMu.Unlock()
	j.stateMu.Lock()
	p, m := j.phase, j.message
	j.stateMu.Unlock()
	switch p {
	case "completed":
		select {
		case ch <- `{"event":"completed"}`:
		default:
		}
	case "failed":
		line, _ := json.Marshal(map[string]string{"event": "failed", "message": strings.TrimSpace(m)})
		select {
		case ch <- string(line):
		default:
		}
	case "running":
		select {
		case ch <- `{"event":"running"}`:
		default:
		}
	default:
		select {
		case ch <- `{"event":"queued"}`:
		default:
		}
	}
	j.streamMu.Lock()
	catch := j.streamAcc.String()
	j.streamMu.Unlock()
	if catch != "" {
		line, err := json.Marshal(map[string]string{"event": "chunk", "content": catch})
		if err == nil {
			select {
			case ch <- string(line):
			default:
			}
		}
	}
	unsub := func() {
		j.subsMu.Lock()
		for i, c := range j.subs {
			if c == ch {
				j.subs = append(j.subs[:i], j.subs[i+1:]...)
				break
			}
		}
		j.subsMu.Unlock()
	}
	return ch, unsub, nil
}

func (h *debateV2JobHub) scheduleRemove(jobID string, after time.Duration) {
	time.AfterFunc(after, func() { h.remove(jobID) })
}

func (h *debateV2JobHub) remove(jobID string) {
	h.mu.Lock()
	j := h.jobs[jobID]
	delete(h.jobs, jobID)
	h.mu.Unlock()
	if j == nil {
		return
	}
	j.subsMu.Lock()
	subs := j.subs
	j.subs = nil
	j.subsMu.Unlock()
	for _, ch := range subs {
		close(ch)
	}
}
