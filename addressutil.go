package nodemailer

import "strings"

// Local returns the local part of the address (everything before the last "@"),
// or the empty string when the address has no "@".
func (a Address) Local() string {
	if at := strings.LastIndex(a.Address, "@"); at >= 0 {
		return a.Address[:at]
	}
	return ""
}

// Domain returns the domain part of the address (everything after the last
// "@"), or the empty string when the address has no "@".
func (a Address) Domain() string {
	if at := strings.LastIndex(a.Address, "@"); at >= 0 && at < len(a.Address)-1 {
		return a.Address[at+1:]
	}
	return ""
}

// Equal reports whether two addresses refer to the same mailbox. The domain is
// compared case-insensitively (domains are case-insensitive per RFC 5321) while
// the local part is compared verbatim. Display names are ignored.
func (a Address) Equal(b Address) bool {
	al, ad := a.Local(), strings.ToLower(a.Domain())
	bl, bd := b.Local(), strings.ToLower(b.Domain())
	return al == bl && ad == bd
}

// NormalizeAddress parses a single address and returns its addr-spec with the
// domain lower-cased and surrounding whitespace removed, discarding any display
// name. It is handy for de-duplicating and comparing recipient addresses. An
// error is returned when the input is not a valid address.
func NormalizeAddress(s string) (string, error) {
	a, err := ParseAddress(s)
	if err != nil {
		return "", err
	}
	return a.Local() + "@" + strings.ToLower(a.Domain()), nil
}

// FormatAddressList renders a slice of addresses as a single RFC 5322 header
// value: a comma-separated list of "Name <addr>" (or bare addr-spec when the
// name is empty), with non-ASCII display names RFC 2047 encoded.
func FormatAddressList(addrs []Address) string {
	parts := make([]string, 0, len(addrs))
	for _, a := range addrs {
		parts = append(parts, a.String())
	}
	return strings.Join(parts, ", ")
}
