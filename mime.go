package inbucket

import (
	"bytes"
	"container/list"
	"fmt"
	"github.com/sloonz/go-qprintable"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
)

type MIMENodeMatcher func(node *MIMENode) bool

type MIMENode struct {
	Parent      *MIMENode
	FirstChild  *MIMENode
	NextSibling *MIMENode
	Type        string
	Content     []byte
}

type MIMEBody struct {
	Text string
	Html string
	Root *MIMENode
}

func NewMIMENode(parent *MIMENode, contentType string) *MIMENode {
	return &MIMENode{Parent: parent, Type: contentType}
}

func (n *MIMENode) BreadthFirstSearch(matcher MIMENodeMatcher) *MIMENode {
	q := list.New()
	q.PushBack(n)

	// Push children onto queue and attempt to match in that order
	for q.Len() > 0 {
		e := q.Front()
		n := e.Value.(*MIMENode)
		if matcher(n) {
			return n
		}
		q.Remove(e)
		c := n.FirstChild
		for c != nil {
			q.PushBack(c)
			c = c.NextSibling
		}
	}

	return nil
}

func (n *MIMENode) String() string {
	children := ""
	siblings := ""
	if n.FirstChild != nil {
		children = n.FirstChild.String()
	}
	if n.NextSibling != nil {
		siblings = n.NextSibling.String()
	}
	return fmt.Sprintf("[%v %v] %v", n.Type, children, siblings)
}

func IsMIMEMessage(mailMsg *mail.Message) bool {
	// Parse top-level multipart
	ctype := mailMsg.Header.Get("Content-Type")
	mediatype, _, err := mime.ParseMediaType(ctype)
	if err != nil {
		return false
	}
	switch mediatype {
	case "multipart/alternative":
		return true
	}

	return false
}

func ParseMIMEBody(mailMsg *mail.Message) (*MIMEBody, error) {
	mimeMsg := new(MIMEBody)

	if !IsMIMEMessage(mailMsg) {
		// Parse as text only
		bodyBytes, err := decodeSection(mailMsg.Header.Get("Content-Transfer-Encoding"), mailMsg.Body)
		if err != nil {
			return nil, err
		}
		mimeMsg.Text = string(bodyBytes)
	} else {
		// Parse top-level multipart
		ctype := mailMsg.Header.Get("Content-Type")
		mediatype, params, err := mime.ParseMediaType(ctype)
		if err != nil {
			return nil, err
		}
		switch mediatype {
		case "multipart/alternative":
			// Good
		default:
			return nil, fmt.Errorf("Unknown mediatype: %v", mediatype)
		}
		boundary := params["boundary"]
		if boundary == "" {
			return nil, fmt.Errorf("Unable to locate boundary param in Content-Type header")
		}

		// Root Node of our tree
		root := NewMIMENode(nil, mediatype)
		err = parseNodes(root, mailMsg.Body, boundary)

		// Locate text body
		match := root.BreadthFirstSearch(func(node *MIMENode) bool {
			return node.Type == "text/plain"
		})
		if match != nil {
			mimeMsg.Text = string(match.Content)
		}

		// Locate HTML body
		match = root.BreadthFirstSearch(func(node *MIMENode) bool {
			return node.Type == "text/html"
		})
		if match != nil {
			mimeMsg.Html = string(match.Content)
		}
	}

	return mimeMsg, nil
}

func (m *MIMEBody) String() string {
	return fmt.Sprintf("----TEXT----\n%v\n----HTML----\n%v\n----END----\n", m.Text, m.Html)
}

// decodeSection attempts to decode the data from reader using the algorithm listed in
// the Content-Transfer-Encoding header, returning the raw data if it does not know
// the encoding type.
func decodeSection(encoding string, reader io.Reader) ([]byte, error) {
	switch strings.ToLower(encoding) {
	case "quoted-printable":
		decoder := qprintable.NewDecoder(qprintable.WindowsTextEncoding, reader)
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(decoder)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	// Don't know this type, just return bytes
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func parseNodes(parent *MIMENode, reader io.Reader, boundary string) error {
	var prevSibling *MIMENode

	// Loop over MIME parts
	mr := multipart.NewReader(reader, boundary)
	for {
		part, err := mr.NextPart()
		if err != nil {
			if err == io.EOF {
				// This is a clean end-of-message signal
				break
			}
			return err
		}
		mediatype, params, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			return err
		}

		// Insert ourselves into tree
		node := NewMIMENode(parent, mediatype)
		if prevSibling != nil {
			prevSibling.NextSibling = node
		} else {
			parent.FirstChild = node
		}
		prevSibling = node

		boundary := params["boundary"]
		if boundary != "" {
			// Content is another multipart
			err = parseNodes(node, part, boundary)
			if err != nil {
				return err
			}
		} else {
			// Content is text or data, decode it
			data, err := decodeSection(part.Header.Get("Content-Transfer-Encoding"), part)
			if err != nil {
				return err
			}
			node.Content = data
		}
	}

	return nil
}
