package inbucket

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
)

const MIME_BUF_BYTES = 1024

type MIMEMessage struct {
	text string
	html string
}

type MIMENode struct {
	Parent      *MIMENode
	FirstChild  *MIMENode
	NextSibling *MIMENode
	Type        string
	Content     *bytes.Buffer
}

func NewMIMENode(parent *MIMENode, contentType string) *MIMENode {
	return &MIMENode{Parent: parent, Type: contentType}
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

func ParseMIMEMessage(mailMsg *mail.Message) (*MIMEMessage, error) {
	mimeMsg := new(MIMEMessage)

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
	fmt.Println(root.String())
	return mimeMsg, err
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
			// Content is data, allocate a buffer
			node.Content = new(bytes.Buffer)
			_, err = io.Copy(node.Content, part)
			if err != nil {
				return err
			}
			fmt.Printf("\n----\n")
			io.Copy(os.Stdout, node.Content)
			fmt.Printf("\n----\n")
		}
	}

	return nil
}
