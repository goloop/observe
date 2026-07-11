package observe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthAlwaysOK(t *testing.T) {
	r := New(WithService("api"), WithBuildInfo(BuildInfo{Version: "v1"}))
	r.Check("down", func(context.Context) error { return errors.New("nope") })

	st := r.Health(context.Background())
	if !st.OK() || st.Service != "api" || st.Version != "v1" {
		t.Fatalf("health: %+v", st)
	}
	if len(st.Checks) != 0 {
		t.Fatalf("health must not run checks: %+v", st.Checks)
	}
}

func TestReadyReflectsChecks(t *testing.T) {
	// Redaction is on by default, so opt out to assert on the error detail.
	r := New(WithErrorRedaction(false))
	r.Check("ok", func(context.Context) error { return nil })
	r.Check("bad", func(context.Context) error { return errors.New("dial timeout") })

	st := r.Ready(context.Background())
	if st.OK() {
		t.Fatal("expected overall error")
	}
	if st.Checks["ok"].Status != "ok" || st.Checks["bad"].Status != "error" {
		t.Fatalf("checks: %+v", st.Checks)
	}
	if st.Checks["bad"].Error != "dial timeout" {
		t.Fatalf("expected error detail, got %q", st.Checks["bad"].Error)
	}
}

func TestReadyRedactsByDefault(t *testing.T) {
	r := New()
	r.Check("bad", func(context.Context) error { return errors.New("host=db user=admin") })

	st := r.Ready(context.Background())
	if st.OK() {
		t.Fatal("expected error status")
	}
	if got := st.Checks["bad"].Error; got != "" {
		t.Fatalf("default must redact error detail, got %q", got)
	}
}

func TestReadyBoundedByHungCheck(t *testing.T) {
	// A check that ignores its context must not block the handler past the
	// per-check timeout: runOne returns ctx.Err() while the check goroutine
	// keeps hanging.
	r := New(WithCheckTimeout(20 * time.Millisecond))
	release := make(chan struct{})
	defer close(release)
	r.Check("hung", func(context.Context) error {
		<-release // ignores ctx entirely
		return nil
	})

	done := make(chan Status, 1)
	go func() { done <- r.Ready(context.Background()) }()
	select {
	case st := <-done:
		if st.OK() {
			t.Fatal("hung check should fail on timeout")
		}
	case <-time.After(time.Second):
		t.Fatal("Ready blocked on a hung check past the per-check timeout")
	}
}

func TestReadyRecoversPanic(t *testing.T) {
	// A panicking check yields an error instead of crashing the process.
	r := New(WithErrorRedaction(false))
	r.Check("boom", func(context.Context) error { panic("kaboom") })

	st := r.Ready(context.Background())
	if st.OK() {
		t.Fatal("panicking check should fail")
	}
	if got := st.Checks["boom"].Error; got == "" {
		t.Fatal("panic should surface as an error detail")
	}
}

func TestCheckIgnoresEmptyName(t *testing.T) {
	r := New()
	r.Check("", func(context.Context) error { return errors.New("down") })
	r.Check("nilfn", nil)

	st := r.Ready(context.Background())
	if !st.OK() {
		t.Fatalf("empty-name and nil checks must be ignored: %+v", st)
	}
	if len(st.Checks) != 0 {
		t.Fatalf("no checks should have registered: %+v", st.Checks)
	}
}

func TestReadyEmptyIsOK(t *testing.T) {
	if st := New().Ready(context.Background()); !st.OK() {
		t.Fatalf("no checks should be ready: %+v", st)
	}
}

func TestErrorRedaction(t *testing.T) {
	r := New(WithErrorRedaction(true))
	r.Check("bad", func(context.Context) error { return errors.New("secret internal detail") })
	st := r.Ready(context.Background())
	if st.OK() {
		t.Fatal("expected error status")
	}
	if st.Checks["bad"].Error != "" {
		t.Fatalf("error must be redacted, got %q", st.Checks["bad"].Error)
	}
}

func TestCheckTimeout(t *testing.T) {
	r := New(WithCheckTimeout(20 * time.Millisecond))
	r.Check("slow", func(ctx context.Context) error {
		select {
		case <-time.After(time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	st := r.Ready(context.Background())
	if st.OK() {
		t.Fatal("slow check should fail on timeout")
	}
}

func TestReadyHandlerStatusCodes(t *testing.T) {
	r := New()
	r.Check("bad", func(context.Context) error { return errors.New("down") })

	rec := httptest.NewRecorder()
	r.ReadyHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	r2 := New()
	rec2 := httptest.NewRecorder()
	r2.ReadyHandler().ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}

	var st Status
	if err := json.NewDecoder(rec2.Body).Decode(&st); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !st.OK() {
		t.Fatalf("body not ok: %+v", st)
	}
}

func TestHealthHandlerAlways200(t *testing.T) {
	r := New()
	r.Check("bad", func(context.Context) error { return errors.New("down") })
	rec := httptest.NewRecorder()
	r.HealthHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("health must be 200 even with a failing check, got %d", rec.Code)
	}
}

func TestConcurrentCheckAndRead(t *testing.T) {
	r := New()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			r.Check("c", func(context.Context) error { return nil })
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		_ = r.Ready(context.Background())
	}
	<-done
}
