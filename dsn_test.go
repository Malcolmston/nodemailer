package nodemailer

import (
	"strings"
	"testing"
)

func TestDSNMailParams(t *testing.T) {
	d := &DSNOptions{Return: DSNReturnFull, EnvID: "abc123"}
	if got := d.mailParams(); got != "RET=FULL ENVID=abc123" {
		t.Errorf("mailParams = %q", got)
	}
	if got := (&DSNOptions{}).mailParams(); got != "" {
		t.Errorf("empty mailParams = %q", got)
	}
	var nilOpts *DSNOptions
	if got := nilOpts.mailParams(); got != "" {
		t.Errorf("nil mailParams = %q", got)
	}
}

func TestDSNRcptParams(t *testing.T) {
	d := &DSNOptions{
		Notify:        []DSNNotify{DSNNotifySuccess, DSNNotifyFailure},
		OrigRecipient: "grace@example.com",
	}
	if got := d.rcptParams(); got != "NOTIFY=SUCCESS,FAILURE ORCPT=rfc822;grace@example.com" {
		t.Errorf("rcptParams = %q", got)
	}
	// Already-prefixed ORCPT is passed through.
	d2 := &DSNOptions{OrigRecipient: "rfc822;a@b.com"}
	if got := d2.rcptParams(); got != "ORCPT=rfc822;a@b.com" {
		t.Errorf("rcptParams = %q", got)
	}
	if !(&DSNOptions{}).empty() {
		t.Error("empty options should report empty")
	}
	if (&DSNOptions{Return: DSNReturnHeaders}).empty() {
		t.Error("options with RET should not be empty")
	}
}

func TestDSNOverSMTP(t *testing.T) {
	srv := newRichServer(t)
	tr := srv.transport(t)
	tr.DSN = &DSNOptions{
		Return:        DSNReturnHeaders,
		EnvID:         "env-42",
		Notify:        []DSNNotify{DSNNotifyFailure, DSNNotifyDelay},
		OrigRecipient: "grace@example.com",
	}
	raw := []byte("Subject: DSN\r\n\r\nbody\r\n")
	if err := tr.Send("ada@example.com", []string{"grace@example.com"}, raw); err != nil {
		t.Fatalf("Send: %v", err)
	}
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if !strings.Contains(srv.lastMailLine, "RET=HDRS") || !strings.Contains(srv.lastMailLine, "ENVID=env-42") {
		t.Errorf("MAIL line missing DSN params: %q", srv.lastMailLine)
	}
	if !strings.Contains(srv.lastRcptLine, "NOTIFY=FAILURE,DELAY") ||
		!strings.Contains(srv.lastRcptLine, "ORCPT=rfc822;grace@example.com") {
		t.Errorf("RCPT line missing DSN params: %q", srv.lastRcptLine)
	}
}
