package nodemailer

import "strings"

// DSNReturn selects how much of the original message a DSN report includes.
type DSNReturn string

const (
	// DSNReturnFull requests the full message be returned (RET=FULL).
	DSNReturnFull DSNReturn = "FULL"
	// DSNReturnHeaders requests only the headers be returned (RET=HDRS).
	DSNReturnHeaders DSNReturn = "HDRS"
)

// DSNNotify selects the conditions under which a delivery status notification
// is generated (RFC 3461 NOTIFY parameter).
type DSNNotify string

const (
	// DSNNotifyNever suppresses all notifications (NOTIFY=NEVER, exclusive).
	DSNNotifyNever DSNNotify = "NEVER"
	// DSNNotifySuccess requests notification on successful delivery.
	DSNNotifySuccess DSNNotify = "SUCCESS"
	// DSNNotifyFailure requests notification on delivery failure.
	DSNNotifyFailure DSNNotify = "FAILURE"
	// DSNNotifyDelay requests notification when delivery is delayed.
	DSNNotifyDelay DSNNotify = "DELAY"
)

// DSNOptions configures RFC 3461 Delivery Status Notifications, mirroring
// nodemailer's dsn option. It influences the SMTP MAIL FROM and RCPT TO
// parameters and has no effect on non-SMTP transports.
type DSNOptions struct {
	// Return controls how much of the message a bounce includes (MAIL RET).
	Return DSNReturn
	// EnvID is an envelope identifier echoed back in the DSN (MAIL ENVID).
	EnvID string
	// Notify lists the conditions that trigger a notification (RCPT NOTIFY).
	Notify []DSNNotify
	// OrigRecipient is the original recipient address recorded in RCPT ORCPT
	// (formatted as "rfc822;<addr>").
	OrigRecipient string
}

// mailParams returns the extra MAIL FROM parameters (e.g. "RET=FULL ENVID=abc"),
// or "" when none apply.
func (d *DSNOptions) mailParams() string {
	if d == nil {
		return ""
	}
	var parts []string
	if d.Return != "" {
		parts = append(parts, "RET="+string(d.Return))
	}
	if d.EnvID != "" {
		parts = append(parts, "ENVID="+d.EnvID)
	}
	return strings.Join(parts, " ")
}

// rcptParams returns the extra RCPT TO parameters (e.g. "NOTIFY=SUCCESS,FAILURE
// ORCPT=rfc822;a@b.com"), or "" when none apply.
func (d *DSNOptions) rcptParams() string {
	if d == nil {
		return ""
	}
	var parts []string
	if len(d.Notify) > 0 {
		notify := make([]string, len(d.Notify))
		for i, n := range d.Notify {
			notify[i] = string(n)
		}
		parts = append(parts, "NOTIFY="+strings.Join(notify, ","))
	}
	if d.OrigRecipient != "" {
		orcpt := d.OrigRecipient
		if !strings.Contains(orcpt, ";") {
			orcpt = "rfc822;" + orcpt
		}
		parts = append(parts, "ORCPT="+orcpt)
	}
	return strings.Join(parts, " ")
}

// empty reports whether no DSN parameters would be emitted.
func (d *DSNOptions) empty() bool {
	return d == nil || (d.mailParams() == "" && d.rcptParams() == "")
}
