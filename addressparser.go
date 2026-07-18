package nodemailer

import (
	"regexp"
	"strings"
)

// ParsedAddr is a single structured entry produced by ParseAddresses. It mirrors
// the object shape returned by nodemailer's lib/addressparser: a leaf entry has
// Name and Address set (with IsGroup false), while a group entry has Name set,
// IsGroup true and Group holding its (flattened) members.
type ParsedAddr struct {
	// Name is the display name (for a leaf) or the group label (for a group).
	Name string
	// Address is the addr-spec for a leaf entry; it is empty for a group.
	Address string
	// Group holds the members of a group entry; it is nil for a leaf.
	Group []ParsedAddr
	// IsGroup reports whether this entry is an RFC 5322 group. When true the
	// entry carries Group (possibly empty) instead of Address.
	IsGroup bool
}

// maxNestedGroupDepth bounds recursion into nested groups, matching nodemailer's
// safeguard against pathological input (RFC 5322 forbids nested groups).
const maxNestedGroupDepth = 50

var (
	// strictEmailRe matches a bare, whitespace-free addr-spec occupying the
	// whole token, e.g. "user@example.com".
	strictEmailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)
	// looseEmailRe matches an addr-spec embedded within free text so it can be
	// lifted out as the address, leaving the remainder as the display name.
	looseEmailRe = regexp.MustCompile(`\s*\b[^@\s]+@[^\s]+\b\s*`)
)

// ParseAddresses parses an RFC 5322 address-field string into structured
// entries, exactly as nodemailer's addressparser does. It is deliberately
// lenient: it accepts display-name-only entries, comments, unterminated
// quotes/brackets, semicolon delimiters and named groups, never rewriting an
// invalid address into a valid one. Quoted local-parts are preserved verbatim
// so that "user@evil.com"@example.com routes to example.com, not evil.com.
//
// Leaf entries carry Name and Address; group entries carry Name, IsGroup and
// Group. Use ParseAddressesFlatten to obtain only the leaves.
func ParseAddresses(str string) []ParsedAddr {
	return parseAddressesDepth(str, false, 0)
}

// ParseAddressesFlatten behaves like ParseAddresses but recursively flattens
// every group so the result contains only leaf entries (Name/Address), matching
// nodemailer's addressparser(str, {flatten:true}).
func ParseAddressesFlatten(str string) []ParsedAddr {
	return parseAddressesDepth(str, true, 0)
}

func parseAddressesDepth(str string, flatten bool, depth int) []ParsedAddr {
	if depth > maxNestedGroupDepth {
		return nil
	}
	tokens := tokenizeAddresses(str)

	// Split the token stream into individual addresses on "," and ";" operators.
	var groups [][]*addrToken
	var cur []*addrToken
	for _, tok := range tokens {
		if tok.typ == addrOperator && (tok.value == "," || tok.value == ";") {
			if len(cur) > 0 {
				groups = append(groups, cur)
			}
			cur = nil
		} else {
			cur = append(cur, tok)
		}
	}
	if len(cur) > 0 {
		groups = append(groups, cur)
	}

	var parsed []ParsedAddr
	for _, g := range groups {
		parsed = append(parsed, handleAddress(g, depth)...)
	}

	// Merge fragments produced when an unquoted display name contains a comma:
	// "Joe Foo, PhD <joe@x>" splits into a name-only entry followed by an entry
	// carrying both name and address; recombine them.
	for i := len(parsed) - 2; i >= 0; i-- {
		curEnt := parsed[i]
		nextEnt := parsed[i+1]
		if curEnt.Address == "" && curEnt.Name != "" && !curEnt.IsGroup && nextEnt.Address != "" && nextEnt.Name != "" {
			parsed[i+1].Name = curEnt.Name + ", " + nextEnt.Name
			parsed = append(parsed[:i], parsed[i+1:]...)
		}
	}

	if flatten {
		var flat []ParsedAddr
		var walk func([]ParsedAddr)
		walk = func(list []ParsedAddr) {
			for _, e := range list {
				if e.IsGroup {
					walk(e.Group)
				} else {
					flat = append(flat, e)
				}
			}
		}
		walk(parsed)
		return flat
	}
	return parsed
}

// handleAddress converts the tokens of a single address into one entry (or, for
// a group, one group entry whose members are recursively parsed and flattened).
func handleAddress(tokens []*addrToken, depth int) []ParsedAddr {
	isGroup := false
	state := "text"
	insideQuotes := false

	data := map[string][]string{
		"address": nil,
		"comment": nil,
		"group":   nil,
		"text":    nil,
	}
	var textWasQuoted []bool
	wasQuoted := func(i int) bool { return i < len(textWasQuoted) && textWasQuoted[i] }

	for i, token := range tokens {
		var prev *addrToken
		if i > 0 {
			prev = tokens[i-1]
		}
		if token.typ == addrOperator {
			switch token.value {
			case "<":
				state = "address"
				insideQuotes = false
			case "(":
				state = "comment"
				insideQuotes = false
			case ":":
				state = "group"
				isGroup = true
				insideQuotes = false
			case `"`:
				insideQuotes = !insideQuotes
				state = "text"
			default:
				state = "text"
				insideQuotes = false
			}
			continue
		}
		if token.value == "" {
			continue
		}
		v := token.value
		if state == "address" {
			// Apple Mail truncates everything before an unexpected "<".
			if idx := strings.IndexByte(v, '<'); idx >= 0 {
				v = strings.TrimLeft(v[idx+1:], " \t\r\n")
			}
		}
		if prev != nil && prev.noBreak && len(data[state]) > 0 {
			data[state][len(data[state])-1] += v
			if state == "text" && insideQuotes && len(textWasQuoted) > 0 {
				textWasQuoted[len(textWasQuoted)-1] = true
			}
		} else {
			data[state] = append(data[state], v)
			if state == "text" {
				textWasQuoted = append(textWasQuoted, insideQuotes)
			}
		}
	}

	// A comment stands in for the display name when no text was seen.
	if len(data["text"]) == 0 && len(data["comment"]) > 0 {
		data["text"] = data["comment"]
		data["comment"] = nil
	}

	if isGroup {
		text := strings.Join(data["text"], " ")
		var members []ParsedAddr
		if len(data["group"]) > 0 {
			for _, m := range parseAddressesDepth(strings.Join(data["group"], ","), false, depth+1) {
				if m.IsGroup {
					members = append(members, m.Group...)
				} else {
					members = append(members, m)
				}
			}
		}
		return []ParsedAddr{{Name: text, Group: members, IsGroup: true}}
	}

	// No explicit <address>: try to recover one from the free text, but never
	// from quoted text (RFC 5321 permits "@" inside a quoted local-part).
	if len(data["address"]) == 0 && len(data["text"]) > 0 {
		for i := len(data["text"]) - 1; i >= 0; i-- {
			if !wasQuoted(i) && strictEmailRe.MatchString(data["text"][i]) {
				data["address"] = []string{data["text"][i]}
				data["text"] = append(data["text"][:i], data["text"][i+1:]...)
				if i < len(textWasQuoted) {
					textWasQuoted = append(textWasQuoted[:i], textWasQuoted[i+1:]...)
				}
				break
			}
		}
		if len(data["address"]) == 0 {
			for i := len(data["text"]) - 1; i >= 0; i-- {
				if wasQuoted(i) {
					continue
				}
				if loc := looseEmailRe.FindStringIndex(data["text"][i]); loc != nil {
					match := data["text"][i][loc[0]:loc[1]]
					data["address"] = []string{strings.TrimSpace(match)}
					data["text"][i] = strings.TrimSpace(data["text"][i][:loc[0]] + " " + data["text"][i][loc[1]:])
					break
				}
			}
		}
	}

	if len(data["text"]) == 0 && len(data["comment"]) > 0 {
		data["text"] = data["comment"]
		data["comment"] = nil
	}

	// Keep only the first addr-spec; demote any extras to display text.
	if len(data["address"]) > 1 {
		data["text"] = append(data["text"], data["address"][1:]...)
		data["address"] = data["address"][:1]
	}

	text := strings.Join(data["text"], " ")
	address := strings.Join(data["address"], " ")

	out := ParsedAddr{Address: address, Name: text}
	if out.Address == "" {
		out.Address = text
	}
	if out.Name == "" {
		out.Name = address
	}
	if out.Address == out.Name {
		if strings.Contains(out.Address, "@") {
			out.Name = ""
		} else {
			out.Address = ""
		}
	}
	return []ParsedAddr{out}
}

// addrTokenType distinguishes operator tokens from text tokens.
type addrTokenType int

const (
	addrText addrTokenType = iota
	addrOperator
)

// addrToken is a single lexical token from an address field.
type addrToken struct {
	typ     addrTokenType
	value   string
	noBreak bool
}

// addrOperatorClose maps each operator character to the character expected to
// close it. A zero rune means the operator is instantaneous (", " and ";").
func addrOperatorClose(chr rune) (rune, bool) {
	switch chr {
	case '"':
		return '"', true
	case '(':
		return ')', true
	case '<':
		return '>', true
	case ',':
		return 0, true
	case ':':
		return ';', true
	case ';':
		return 0, true
	}
	return 0, false
}

func isAddrBreak(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n', ',', ';':
		return true
	}
	return false
}

// tokenizeAddresses splits an address field into operator and text tokens using
// the same state machine as nodemailer's Tokenizer, including domain-literal and
// quoted-string handling.
func tokenizeAddresses(str string) []*addrToken {
	runes := []rune(str)
	var list []*addrToken
	var node *addrToken
	var operatorExpecting rune // 0 == none
	escaped := false
	inDomainLiteral := false

	for i := 0; i < len(runes); i++ {
		chr := runes[i]
		var nextChr rune
		if i < len(runes)-1 {
			nextChr = runes[i+1]
		}

		if !escaped && operatorExpecting == 0 {
			if !inDomainLiteral && chr == '[' {
				inDomainLiteral = true
			} else if inDomainLiteral && (chr == ']' || chr == ',' || chr == ';') {
				inDomainLiteral = false
			}
		}

		switch {
		case escaped:
			// handled below by appending to text
		case operatorExpecting != 0 && chr == operatorExpecting:
			node = &addrToken{typ: addrOperator, value: string(chr)}
			if nextChr != 0 && !isAddrBreak(nextChr) {
				node.noBreak = true
			}
			list = append(list, node)
			node = nil
			operatorExpecting = 0
			escaped = false
			continue
		case operatorExpecting == 0 && !inDomainLiteral && isAddrOperatorStart(chr):
			node = &addrToken{typ: addrOperator, value: string(chr)}
			list = append(list, node)
			node = nil
			operatorExpecting, _ = addrOperatorClose(chr)
			escaped = false
			continue
		case (operatorExpecting == '"' || operatorExpecting == '\'') && chr == '\\':
			escaped = true
			continue
		}

		if node == nil {
			node = &addrToken{typ: addrText}
			list = append(list, node)
		}
		if chr == '\n' {
			chr = ' '
		}
		if chr >= 0x21 || chr == ' ' || chr == '\t' {
			node.value += string(chr)
		}
		escaped = false
	}

	out := make([]*addrToken, 0, len(list))
	for _, n := range list {
		n.value = strings.TrimSpace(n.value)
		if n.value != "" {
			out = append(out, n)
		}
	}
	return out
}

func isAddrOperatorStart(chr rune) bool {
	_, ok := addrOperatorClose(chr)
	return ok
}
