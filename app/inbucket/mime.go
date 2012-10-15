package inbucket

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
)

const MIME_BUF_BYTES = 1024

type MIMEMessage struct {
	text string
	html string
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

	// Buffer used by Part.Read to hand us chunks of data
	readBytes := make([]byte, MIME_BUF_BYTES)

	// Loop over MIME parts
	mr := multipart.NewReader(mailMsg.Body, boundary)
	for {
		part, err := mr.NextPart()
		if err != nil {
			if err == io.EOF {
				// This is a clean end-of-message signal
				break
			}
			return nil, err
		}
		mediatype, params, err = mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			return nil, err
		}
		if mediatype == "text/plain" && mimeMsg.text == "" {
			// First text section, we'll have that, but first we have to deal with
			// the odd way Part lets us read data.
			var buf bytes.Buffer
			for {
				n, err := part.Read(readBytes)
				if err != nil {
					if err == io.EOF {
						// Clean end of part signal
						break
					}
					return nil, err
				}
				// Extra data in readBytes is not cleared, so we must respect
				// the value returned in 'n'
				buf.Write(readBytes[:n])
			}

			mimeMsg.text = buf.String()
			fmt.Printf("text: '%v'\n", mimeMsg.text)
		}
	}

	return mimeMsg, nil
}
