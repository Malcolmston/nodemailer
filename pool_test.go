package nodemailer

import (
	"strings"
	"sync"
	"testing"
)

func TestVerifySuccess(t *testing.T) {
	srv := newRichServer(t)
	tr := srv.transport(t)
	if err := tr.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVerifyWithAuth(t *testing.T) {
	srv := newRichServer(t)
	srv.advertiseAuth = true
	tr := srv.transport(t)
	tr.Auth = XOAuth2Auth("ada@example.com", "tok")
	if err := tr.Verify(); err != nil {
		t.Fatalf("Verify with auth: %v", err)
	}
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.authUser != "ada@example.com" {
		t.Errorf("auth not performed during verify: %q", srv.authUser)
	}
}

func TestVerifyNoHost(t *testing.T) {
	if err := (&SMTPTransport{}).Verify(); err == nil {
		t.Error("expected error for missing host")
	}
}

func TestPoolReusesConnections(t *testing.T) {
	srv := newRichServer(t)
	pool := &Pool{Transport: srv.transport(t), MaxConnections: 2, MaxMessages: 100}
	defer func() { _ = pool.Close() }()

	// Send several messages sequentially; they should share a single connection.
	for i := 0; i < 5; i++ {
		if err := pool.Send("a@b.com", []string{"c@d.com"}, []byte("Subject: x\r\n\r\nhi\r\n")); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	srv.mu.Lock()
	conns, msgs := srv.conns, srv.messages
	srv.mu.Unlock()
	if msgs != 5 {
		t.Errorf("server accepted %d messages, want 5", msgs)
	}
	if conns != 1 {
		t.Errorf("sequential sends opened %d connections, want 1 (reuse)", conns)
	}
}

func TestPoolRecyclesAtMaxMessages(t *testing.T) {
	srv := newRichServer(t)
	pool := &Pool{Transport: srv.transport(t), MaxConnections: 1, MaxMessages: 2}
	defer func() { _ = pool.Close() }()

	for i := 0; i < 4; i++ {
		if err := pool.Send("a@b.com", []string{"c@d.com"}, []byte("Subject: x\r\n\r\nhi\r\n")); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	srv.mu.Lock()
	conns := srv.conns
	srv.mu.Unlock()
	// With MaxMessages=2 and 4 messages, the pool recycles once: 2 connections.
	if conns != 2 {
		t.Errorf("opened %d connections, want 2 (recycle at MaxMessages)", conns)
	}
}

func TestPoolConcurrent(t *testing.T) {
	srv := newRichServer(t)
	pool := &Pool{Transport: srv.transport(t), MaxConnections: 3}
	defer func() { _ = pool.Close() }()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = pool.Send("a@b.com", []string{"c@d.com"}, []byte("Subject: x\r\n\r\nhi\r\n"))
		}()
	}
	wg.Wait()

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.messages != 20 {
		t.Errorf("accepted %d messages, want 20", srv.messages)
	}
	if srv.conns > 3 {
		t.Errorf("opened %d connections, want <= MaxConnections (3)", srv.conns)
	}
}

func TestPoolClosedAndErrors(t *testing.T) {
	if err := (&Pool{}).Send("a@b.com", nil, nil); err == nil {
		t.Error("expected error with nil Transport")
	}
	srv := newRichServer(t)
	pool := &Pool{Transport: srv.transport(t)}
	if err := pool.Verify(); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := pool.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := pool.Close(); err != nil {
		t.Errorf("second Close should be nil, got %v", err)
	}
	if err := pool.Send("a@b.com", []string{"c@d.com"}, []byte("x")); err != ErrPoolClosed {
		t.Errorf("send after close = %v, want ErrPoolClosed", err)
	}
	if err := (&Pool{}).Verify(); err == nil {
		t.Error("expected verify error with nil Transport")
	}
}

func TestPoolDelivers(t *testing.T) {
	srv := newRichServer(t)
	pool := &Pool{Transport: srv.transport(t)}
	defer func() { _ = pool.Close() }()
	if err := pool.Send("a@b.com", []string{"c@d.com"}, []byte("Subject: Pooled\r\n\r\nbody\r\n")); err != nil {
		t.Fatal(err)
	}
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.datas) != 1 || !strings.Contains(srv.datas[0], "Subject: Pooled") {
		t.Errorf("unexpected data: %v", srv.datas)
	}
}
