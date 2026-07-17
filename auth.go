package nodemailer

import (
	"encoding/base64"
	"net/smtp"
)

// XOAuth2Auth returns an smtp.Auth that implements the SASL XOAUTH2 mechanism
// used by Gmail, Outlook and other OAuth2-enabled SMTP servers.
//
// user is the account's email address and accessToken is a valid OAuth2 bearer
// access token (not the refresh token). The mechanism sends a single initial
// client response and, on failure, answers the server's error challenge with an
// empty line so the session can be torn down cleanly.
func XOAuth2Auth(user, accessToken string) smtp.Auth {
	return &xoauth2Auth{user: user, token: accessToken}
}

// xoauth2Auth is the SASL XOAUTH2 mechanism.
type xoauth2Auth struct {
	user  string
	token string
}

// XOAuth2Token builds the raw (un-encoded) XOAUTH2 client response string:
//
//	user=<user>^Aauth=Bearer <token>^A^A
//
// where ^A is the ASCII SOH (0x01) separator. It is exported so callers can
// construct the token independently, for example when driving a custom client.
func XOAuth2Token(user, accessToken string) string {
	return "user=" + user + "\x01auth=Bearer " + accessToken + "\x01\x01"
}

// Start begins the XOAUTH2 exchange, returning the mechanism name and the
// base64-decoded initial client response. net/smtp base64-encodes the returned
// bytes before transmission.
func (a *xoauth2Auth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "XOAUTH2", []byte(XOAuth2Token(a.user, a.token)), nil
}

// Next handles server continuations. A well-behaved server accepts the initial
// response and never calls Next with more; if authentication fails the server
// sends a base64 error challenge, to which the client must reply with an empty
// line before the server reports the final failure status.
func (a *xoauth2Auth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		// The server rejected the token and sent a base64-encoded JSON error.
		// Acknowledge with an empty response so it can return its failure code.
		return []byte{}, nil
	}
	return nil, nil
}

// decodeXOAuth2 parses a raw XOAUTH2 client response back into its user and
// bearer token. It is used by the in-process test SMTP server to verify tokens.
func decodeXOAuth2(resp []byte) (user, token string, ok bool) {
	// resp may be base64 (as seen on the wire) or already decoded.
	if dec, err := base64.StdEncoding.DecodeString(string(resp)); err == nil {
		if len(dec) > 0 && dec[0] == 'u' {
			resp = dec
		}
	}
	s := string(resp)
	var gotUser, gotToken bool
	for _, field := range splitSOH(s) {
		switch {
		case len(field) > 5 && field[:5] == "user=":
			user = field[5:]
			gotUser = true
		case len(field) > 12 && field[:12] == "auth=Bearer ":
			token = field[12:]
			gotToken = true
		}
	}
	return user, token, gotUser && gotToken
}

// splitSOH splits s on ASCII SOH (0x01), dropping empty trailing fields.
func splitSOH(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\x01' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// ensure the interface is satisfied at compile time.
var _ smtp.Auth = (*xoauth2Auth)(nil)
