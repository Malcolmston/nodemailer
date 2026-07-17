package nodemailer

import (
	"bytes"
	"fmt"
	"os/exec"
)

// execCommand builds the sendmail command. It is a variable so tests can
// substitute a fake process.
var execCommand = exec.Command

// SendmailTransport delivers messages by piping them to a local sendmail-compatible
// binary via os/exec, mirroring nodemailer's sendmail transport.
type SendmailTransport struct {
	// Path is the sendmail binary to run. It defaults to "/usr/sbin/sendmail".
	Path string
	// Args are extra arguments inserted before the computed envelope arguments.
	Args []string
	// UseHeaders passes -t so sendmail extracts recipients from the message
	// headers instead of the command line. When false the envelope sender (-f)
	// and the recipient addresses are passed explicitly.
	UseHeaders bool
}

// path returns the configured binary path or the default.
func (t *SendmailTransport) path() string {
	if t.Path != "" {
		return t.Path
	}
	return "/usr/sbin/sendmail"
}

// args computes the full argument list for a given envelope.
func (t *SendmailTransport) args(from string, to []string) []string {
	args := append([]string{"-i"}, t.Args...)
	if from != "" {
		args = append(args, "-f", from)
	}
	if t.UseHeaders {
		args = append(args, "-t")
		return args
	}
	args = append(args, "--")
	return append(args, to...)
}

// Send pipes the raw message to the sendmail binary's standard input.
func (t *SendmailTransport) Send(from string, to []string, raw []byte) error {
	cmd := execCommand(t.path(), t.args(from, to)...)
	cmd.Stdin = bytes.NewReader(raw)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("nodemailer: sendmail: %w: %s", err, stderr.String())
		}
		return fmt.Errorf("nodemailer: sendmail: %w", err)
	}
	return nil
}
