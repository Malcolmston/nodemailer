package nodemailer

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	stdpath "path"
	"path/filepath"
	"strings"
	"time"
)

// httpClient is the HTTP client used to fetch URL attachments. It is a variable
// so tests can substitute a client backed by an in-process server.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// AttachFile reads the file at path and appends it as a regular attachment. The
// filename defaults to the base name of path and the content type is guessed
// from the extension, falling back to content sniffing.
func (m *Message) AttachFile(path string) *Message {
	a, err := attachmentFromFile(path, false)
	if err != nil {
		m.setErr(err)
		return m
	}
	return m.Attach(a)
}

// AttachReader reads all bytes from r and appends them as a regular attachment
// with the given filename and content type. If contentType is empty it is
// sniffed from the content and filename.
func (m *Message) AttachReader(filename, contentType string, r io.Reader) *Message {
	data, err := io.ReadAll(r)
	if err != nil {
		m.setErr(fmt.Errorf("nodemailer: read attachment %q: %w", filename, err))
		return m
	}
	return m.Attach(Attachment{
		Filename:    filename,
		Content:     data,
		ContentType: sniffContentType(contentType, filename, data),
	})
}

// AttachURL fetches the resource at url over HTTP(S) and appends it as a regular
// attachment. The filename defaults to the last path segment of the URL and the
// content type is taken from the response, then sniffed as a fallback.
func (m *Message) AttachURL(url string) *Message {
	a, err := attachmentFromURL(url)
	if err != nil {
		m.setErr(err)
		return m
	}
	return m.Attach(a)
}

// EmbedFile reads the file at path and appends it as an inline resource
// referenceable from HTML via cid:contentID.
func (m *Message) EmbedFile(contentID, path string) *Message {
	a, err := attachmentFromFile(path, true)
	if err != nil {
		m.setErr(err)
		return m
	}
	a.ContentID = contentID
	return m.Attach(a)
}

// EmbedReader reads all bytes from r and appends them as an inline resource
// referenceable via cid:contentID.
func (m *Message) EmbedReader(contentID, filename, contentType string, r io.Reader) *Message {
	data, err := io.ReadAll(r)
	if err != nil {
		m.setErr(fmt.Errorf("nodemailer: read embed %q: %w", filename, err))
		return m
	}
	return m.Attach(Attachment{
		Filename:    filename,
		Content:     data,
		ContentType: sniffContentType(contentType, filename, data),
		ContentID:   contentID,
	})
}

// attachmentFromFile builds an Attachment from a filesystem path.
func attachmentFromFile(path string, inline bool) (Attachment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Attachment{}, fmt.Errorf("nodemailer: read file %q: %w", path, err)
	}
	name := filepath.Base(path)
	return Attachment{
		Filename:    name,
		Content:     data,
		ContentType: sniffContentType("", name, data),
		Inline:      inline,
	}, nil
}

// attachmentFromURL builds an Attachment by fetching a URL.
func attachmentFromURL(url string) (Attachment, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return Attachment{}, fmt.Errorf("nodemailer: fetch %q: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Attachment{}, fmt.Errorf("nodemailer: fetch %q: unexpected status %s", url, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Attachment{}, fmt.Errorf("nodemailer: read %q: %w", url, err)
	}
	name := urlFilename(url)
	ct := resp.Header.Get("Content-Type")
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	return Attachment{
		Filename:    name,
		Content:     data,
		ContentType: sniffContentType(ct, name, data),
	}, nil
}

// urlFilename derives a filename from the final path segment of a URL.
func urlFilename(url string) string {
	u := url
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	u = strings.TrimRight(u, "/")
	base := stdpath.Base(u)
	if base == "" || base == "." || base == "/" {
		return "attachment"
	}
	return base
}

// sniffContentType resolves a content type, preferring an explicit value, then
// the filename extension, then HTTP content sniffing of the data.
func sniffContentType(explicit, filename string, data []byte) string {
	if explicit != "" && explicit != "application/octet-stream" {
		return explicit
	}
	if ext := stdpath.Ext(filename); ext != "" {
		if byExt := extType(ext); byExt != "" {
			return byExt
		}
	}
	if len(data) > 0 {
		if sniffed := http.DetectContentType(data); sniffed != "application/octet-stream" {
			return sniffed
		}
	}
	if explicit != "" {
		return explicit
	}
	return "application/octet-stream"
}

// extType returns the MIME type registered for a file extension, stripped of
// any charset or other parameters.
func extType(ext string) string {
	ct := mime.TypeByExtension(ext)
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	return ct
}
