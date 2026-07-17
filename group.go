package nodemailer

import (
	"fmt"
	"net/mail"
	"strings"
)

// AddressGroup is a named RFC 5322 address group, e.g.
//
//	Team: ada@example.com, grace@example.com;
//
// Name is the group's display label and Addresses are its members (which may be
// empty for an "undisclosed recipients:;" style group).
type AddressGroup struct {
	// Name is the group label shown before the colon.
	Name string
	// Addresses are the group members.
	Addresses []Address
}

// ParseAddressGroup parses a single RFC 5322 group construct of the form
// "Name: addr, addr;". The trailing semicolon is optional. It returns an error
// if the input is not a group (no colon) or any member is invalid.
func ParseAddressGroup(s string) (AddressGroup, error) {
	s = strings.TrimSpace(s)
	colon := strings.IndexByte(s, ':')
	if colon < 0 {
		return AddressGroup{}, fmt.Errorf("%w: not a group (missing ':') in %q", ErrInvalidAddress, s)
	}
	name := strings.TrimSpace(s[:colon])
	rest := strings.TrimSpace(s[colon+1:])
	rest = strings.TrimSuffix(rest, ";")
	rest = strings.TrimSpace(rest)

	g := AddressGroup{Name: name}
	if rest == "" {
		return g, nil
	}
	list, err := mail.ParseAddressList(rest)
	if err != nil {
		return AddressGroup{}, fmt.Errorf("%w: %q: %v", ErrInvalidAddress, s, err)
	}
	for _, a := range list {
		addr := Address{Name: a.Name, Address: a.Address}
		if err := addr.Validate(); err != nil {
			return AddressGroup{}, err
		}
		g.Addresses = append(g.Addresses, addr)
	}
	return g, nil
}

// String renders the group as an RFC 5322 header value:
// "Name: a@x, b@y;". A group with no members renders as "Name:;".
func (g AddressGroup) String() string {
	parts := make([]string, len(g.Addresses))
	for i, a := range g.Addresses {
		parts[i] = a.String()
	}
	return g.Name + ": " + strings.Join(parts, ", ") + ";"
}

// AddToGroup appends a named group of recipients to the To header. The members
// become envelope recipients and are rendered as an RFC 5322 group in the To
// header. Members are parsed and validated; the first error is deferred to
// Build.
func (m *Message) AddToGroup(name string, addrs ...string) *Message {
	return m.addGroup(&m.ToGroups, name, addrs)
}

// AddCcGroup appends a named group of recipients to the Cc header.
func (m *Message) AddCcGroup(name string, addrs ...string) *Message {
	return m.addGroup(&m.CcGroups, name, addrs)
}

// addGroup parses the members and records the group for header rendering and
// envelope expansion.
func (m *Message) addGroup(groups *[]AddressGroup, name string, addrs []string) *Message {
	g := AddressGroup{Name: name}
	for _, s := range addrs {
		list, err := ParseAddressList(s)
		if err != nil {
			m.setErr(err)
			return m
		}
		g.Addresses = append(g.Addresses, list...)
	}
	*groups = append(*groups, g)
	return m
}

// groupAddresses flattens the members of a slice of groups.
func groupAddresses(groups []AddressGroup) []Address {
	var out []Address
	for _, g := range groups {
		out = append(out, g.Addresses...)
	}
	return out
}

// recipientHeader renders a To/Cc header value combining flat addresses and any
// named groups. It returns "" when there is nothing to render.
func recipientHeader(flat []Address, groups []AddressGroup) string {
	var parts []string
	if len(flat) > 0 {
		parts = append(parts, addressListString(flat))
	}
	for _, g := range groups {
		parts = append(parts, g.String())
	}
	return strings.Join(parts, ", ")
}
