package nodemailer

import (
	"errors"
	"net/smtp"
	"sync"
)

// Pool is a bounded pool of reusable SMTP connections, mirroring nodemailer's
// pooled SMTP transport. It satisfies the Transport interface, dialing lazily
// and recycling connections after a configurable number of messages.
//
// A Pool is safe for concurrent use. Callers must invoke Close when finished to
// release the underlying connections.
type Pool struct {
	// Transport describes how to reach the SMTP server. It is required.
	Transport *SMTPTransport
	// MaxConnections bounds the number of simultaneously open connections. It
	// defaults to 5.
	MaxConnections int
	// MaxMessages is the number of messages a single connection may carry before
	// it is closed and replaced. It defaults to 100.
	MaxMessages int

	once   sync.Once
	sem    chan struct{}
	idle   chan *pooledConn
	mu     sync.Mutex
	closed bool
}

// pooledConn wraps an SMTP client with a per-connection message counter.
type pooledConn struct {
	client *smtp.Client
	count  int
}

// ErrPoolClosed is returned when a Pool is used after Close.
var ErrPoolClosed = errors.New("nodemailer: pool is closed")

// init lazily sizes the pool's channels the first time it is used.
func (p *Pool) init() {
	p.once.Do(func() {
		if p.MaxConnections <= 0 {
			p.MaxConnections = 5
		}
		if p.MaxMessages <= 0 {
			p.MaxMessages = 100
		}
		p.sem = make(chan struct{}, p.MaxConnections)
		p.idle = make(chan *pooledConn, p.MaxConnections)
	})
}

// Send delivers a message, reusing an idle connection when one is available and
// otherwise opening a new one (subject to MaxConnections).
func (p *Pool) Send(from string, to []string, raw []byte) error {
	p.init()
	if p.Transport == nil {
		return errors.New("nodemailer: Pool requires a Transport")
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrPoolClosed
	}
	p.mu.Unlock()

	// Bound the number of live connections.
	p.sem <- struct{}{}
	defer func() { <-p.sem }()

	pc, err := p.acquire()
	if err != nil {
		return err
	}

	if err := p.Transport.deliver(pc.client, from, to, raw); err != nil {
		_ = pc.client.Close()
		return err
	}
	pc.count++
	p.release(pc)
	return nil
}

// acquire returns an idle connection ready for a new transaction, or dials a
// fresh one. A reused connection is reset (RSET) to clear any prior state.
func (p *Pool) acquire() (*pooledConn, error) {
	select {
	case pc := <-p.idle:
		if err := pc.client.Reset(); err != nil {
			_ = pc.client.Close()
			return p.dial()
		}
		return pc, nil
	default:
		return p.dial()
	}
}

// dial opens a new authenticated connection.
func (p *Pool) dial() (*pooledConn, error) {
	client, err := p.Transport.newClient()
	if err != nil {
		return nil, err
	}
	return &pooledConn{client: client}, nil
}

// release returns a connection to the idle set, or closes it when it has
// reached MaxMessages, the pool is closed or the idle set is full.
func (p *Pool) release(pc *pooledConn) {
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	if closed || pc.count >= p.MaxMessages {
		_ = pc.client.Quit()
		return
	}
	select {
	case p.idle <- pc:
	default:
		_ = pc.client.Quit()
	}
}

// Verify checks connectivity and authentication without sending a message.
func (p *Pool) Verify() error {
	if p.Transport == nil {
		return errors.New("nodemailer: Pool requires a Transport")
	}
	return p.Transport.Verify()
}

// Close closes all idle connections and marks the pool closed. In-flight sends
// close their connections as they complete. Close is idempotent.
func (p *Pool) Close() error {
	p.init()
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	for {
		select {
		case pc := <-p.idle:
			_ = pc.client.Quit()
		default:
			return nil
		}
	}
}
