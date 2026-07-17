package nodemailer

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

// Address represents a single email address with an optional display name,
// e.g. Name="Ada Lovelace", Address="ada@example.com".
type Address struct {
	// Name is the optional human-readable display name.
	Name string
	// Address is the addr-spec, e.g. "ada@example.com".
	Address string
}

// ErrInvalidAddress is returned when an address fails validation.
var ErrInvalidAddress = errors.New("nodemailer: invalid email address")

// ParseAddress parses a single RFC 5322 address such as
// "Ada Lovelace <ada@example.com>" or "ada@example.com".
func ParseAddress(s string) (Address, error) {
	a, err := mail.ParseAddress(s)
	if err != nil {
		return Address{}, fmt.Errorf("%w: %q: %v", ErrInvalidAddress, s, err)
	}
	addr := Address{Name: a.Name, Address: a.Address}
	if err := addr.Validate(); err != nil {
		return Address{}, err
	}
	return addr, nil
}

// ParseAddressList parses a comma-separated list of addresses such as
// "Ada <ada@example.com>, grace@example.com".
func ParseAddressList(s string) ([]Address, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	list, err := mail.ParseAddressList(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %q: %v", ErrInvalidAddress, s, err)
	}
	out := make([]Address, 0, len(list))
	for _, a := range list {
		addr := Address{Name: a.Name, Address: a.Address}
		if err := addr.Validate(); err != nil {
			return nil, err
		}
		out = append(out, addr)
	}
	return out, nil
}

// Validate reports whether the address is a plausible, well-formed addr-spec.
// It rejects clearly-invalid values: empty addresses, addresses without a
// single "@", empty local or domain parts, whitespace inside the addr-spec and
// domains without a dot.
func (a Address) Validate() error {
	addr := strings.TrimSpace(a.Address)
	if addr == "" {
		return fmt.Errorf("%w: empty address", ErrInvalidAddress)
	}
	if strings.ContainsAny(addr, " \t\r\n") {
		return fmt.Errorf("%w: whitespace in %q", ErrInvalidAddress, addr)
	}
	at := strings.LastIndex(addr, "@")
	if at <= 0 || at == len(addr)-1 {
		return fmt.Errorf("%w: missing local or domain part in %q", ErrInvalidAddress, addr)
	}
	local, domain := addr[:at], addr[at+1:]
	if strings.Contains(local, "@") {
		return fmt.Errorf("%w: multiple @ in %q", ErrInvalidAddress, addr)
	}
	if !strings.Contains(domain, ".") {
		return fmt.Errorf("%w: domain %q has no dot", ErrInvalidAddress, domain)
	}
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return fmt.Errorf("%w: malformed domain %q", ErrInvalidAddress, domain)
	}
	return nil
}

// String renders the address for a header, applying RFC 2047 encoded-word
// encoding to non-ASCII display names and quoting when required.
func (a Address) String() string {
	m := &mail.Address{Name: a.Name, Address: a.Address}
	return m.String()
}

// addressListString renders a slice of addresses as a comma-separated header
// value suitable for To/Cc/Bcc/Reply-To.
func addressListString(list []Address) string {
	parts := make([]string, len(list))
	for i, a := range list {
		parts[i] = a.String()
	}
	return strings.Join(parts, ", ")
}
