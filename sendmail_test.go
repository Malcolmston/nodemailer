package nodemailer

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestHelperProcess is not a real test; it is re-executed as a fake sendmail
// binary by the sendmail tests. It echoes its arguments and stdin to files named
// by the NM_ARGS_FILE / NM_STDIN_FILE environment variables, then exits with the
// code in NM_EXIT.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	if f := os.Getenv("NM_ARGS_FILE"); f != "" {
		_ = os.WriteFile(f, []byte(strings.Join(os.Args[3:], "\n")), 0o644)
	}
	if f := os.Getenv("NM_STDIN_FILE"); f != "" {
		data, _ := os.ReadFile("/dev/stdin")
		_ = os.WriteFile(f, data, 0o644)
	}
	if os.Getenv("NM_EXIT") == "1" {
		_, _ = os.Stderr.WriteString("sendmail: simulated failure")
		os.Exit(1)
	}
	os.Exit(0)
}

// fakeSendmail returns an execCommand replacement that re-invokes the test
// binary's TestHelperProcess as a stand-in sendmail.
func fakeSendmail(argsFile, stdinFile, exit string) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], append([]string{"-test.run=TestHelperProcess", "--"}, args...)...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"NM_ARGS_FILE="+argsFile,
			"NM_STDIN_FILE="+stdinFile,
			"NM_EXIT="+exit,
		)
		return cmd
	}
}

func TestSendmailTransport(t *testing.T) {
	dir := t.TempDir()
	argsFile := dir + "/args"
	stdinFile := dir + "/stdin"

	orig := execCommand
	execCommand = fakeSendmail(argsFile, stdinFile, "0")
	defer func() { execCommand = orig }()

	tr := &SendmailTransport{Path: "/usr/sbin/sendmail"}
	raw := []byte("Subject: Hi\r\n\r\nbody\r\n")
	if err := tr.Send("ada@example.com", []string{"grace@example.com", "carl@example.com"}, raw); err != nil {
		t.Fatalf("Send: %v", err)
	}

	args, _ := os.ReadFile(argsFile)
	argStr := string(args)
	for _, want := range []string{"-i", "-f", "ada@example.com", "--", "grace@example.com", "carl@example.com"} {
		if !strings.Contains(argStr, want) {
			t.Errorf("args missing %q; got %q", want, argStr)
		}
	}
	stdin, _ := os.ReadFile(stdinFile)
	if string(stdin) != string(raw) {
		t.Errorf("stdin = %q, want %q", stdin, raw)
	}
}

func TestSendmailUseHeaders(t *testing.T) {
	dir := t.TempDir()
	argsFile := dir + "/args"

	orig := execCommand
	execCommand = fakeSendmail(argsFile, dir+"/stdin", "0")
	defer func() { execCommand = orig }()

	tr := &SendmailTransport{UseHeaders: true}
	if err := tr.Send("ada@example.com", []string{"grace@example.com"}, []byte("hi")); err != nil {
		t.Fatal(err)
	}
	args, _ := os.ReadFile(argsFile)
	argStr := string(args)
	if !strings.Contains(argStr, "-t") {
		t.Errorf("expected -t flag; got %q", argStr)
	}
	if strings.Contains(argStr, "grace@example.com") {
		t.Errorf("recipients should not be on command line with -t; got %q", argStr)
	}
}

func TestSendmailFailure(t *testing.T) {
	dir := t.TempDir()
	orig := execCommand
	execCommand = fakeSendmail(dir+"/args", dir+"/stdin", "1")
	defer func() { execCommand = orig }()

	tr := &SendmailTransport{}
	err := tr.Send("a@b.com", []string{"c@d.com"}, []byte("x"))
	if err == nil {
		t.Fatal("expected sendmail failure")
	}
	if !strings.Contains(err.Error(), "simulated failure") {
		t.Errorf("error missing stderr: %v", err)
	}
}

func TestSendmailDefaultPath(t *testing.T) {
	tr := &SendmailTransport{}
	if tr.path() != "/usr/sbin/sendmail" {
		t.Errorf("default path = %q", tr.path())
	}
	tr.Path = "/custom/sendmail"
	if tr.path() != "/custom/sendmail" {
		t.Errorf("custom path = %q", tr.path())
	}
}

func TestSendmailArgs(t *testing.T) {
	tr := &SendmailTransport{Args: []string{"-N", "never"}}
	args := tr.args("from@x.com", []string{"a@y.com"})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-N never") || !strings.Contains(joined, "-f from@x.com") {
		t.Errorf("args = %v", args)
	}
}
