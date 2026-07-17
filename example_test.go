package nodemailer_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/malcolmston/nodemailer"
)

// Example builds a deterministic message and prints its structural headers.
func Example() {
	msg := nodemailer.New().
		SetFrom("Ada Lovelace <ada@example.com>").
		AddTo("grace@example.com").
		SetSubject("Progress report").
		SetText("Plain-text body").
		SetHTML("<p>HTML body</p>").
		SetDate(time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)).
		SetMessageID("report-42@example.com").
		SetBoundary("EXAMPLE_BOUNDARY")

	raw, err := msg.Build()
	if err != nil {
		panic(err)
	}

	for _, line := range strings.Split(string(raw), "\r\n") {
		switch {
		case strings.HasPrefix(line, "Subject:"),
			strings.HasPrefix(line, "Message-ID:"),
			strings.HasPrefix(line, "Content-Type: multipart"):
			fmt.Println(line)
		}
	}

	// Output:
	// Message-ID: <report-42@example.com>
	// Subject: Progress report
	// Content-Type: multipart/alternative; boundary="EXAMPLE_BOUNDARY"
}

// ExampleTransporter_SendMail sends a message through the in-memory transport
// and prints the resulting envelope.
func ExampleTransporter_SendMail() {
	tr := nodemailer.NewTransporter(&nodemailer.MemoryTransport{})
	msg := nodemailer.New().
		SetFrom("ada@example.com").
		AddTo("grace@example.com").
		AddCc("carl@example.com").
		SetSubject("Welcome").
		SetText("Hi!").
		SetDate(time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)).
		SetMessageID("welcome-1@example.com").
		SetBoundary("B")
	info, err := tr.SendMail(msg)
	if err != nil {
		panic(err)
	}
	fmt.Println(info.Envelope.From)
	fmt.Println(strings.Join(info.Envelope.To, ", "))
	fmt.Println(info.MessageID)

	// Output:
	// ada@example.com
	// grace@example.com, carl@example.com
	// welcome-1@example.com
}
