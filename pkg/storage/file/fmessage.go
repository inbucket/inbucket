package file

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/mail"
	"os"
	"path/filepath"
	"time"

	"github.com/jhillyerd/enmime"
	"github.com/jhillyerd/inbucket/pkg/log"
	"github.com/jhillyerd/inbucket/pkg/storage"
)

// Message implements Message and contains a little bit of data about a
// particular email message, and methods to retrieve the rest of it from disk.
type Message struct {
	mailbox *mbox
	// Stored in GOB
	Fid      string
	Fdate    time.Time
	Ffrom    *mail.Address
	Fto      []*mail.Address
	Fsubject string
	Fsize    int64
	// These are for creating new messages only
	writable   bool
	writerFile *os.File
	writer     *bufio.Writer
}

// newMessage creates a new FileMessage object and sets the Date and ID fields.
// It will also delete messages over messageCap if configured.
func (mb *mbox) newMessage() (storage.StoreMessage, error) {
	// Load index
	if !mb.indexLoaded {
		if err := mb.readIndex(); err != nil {
			return nil, err
		}
	}
	// Delete old messages over messageCap
	if mb.store.messageCap > 0 {
		for len(mb.messages) >= mb.store.messageCap {
			log.Infof("Mailbox %q over configured message cap", mb.name)
			if err := mb.removeMessage(mb.messages[0].ID()); err != nil {
				log.Errorf("Error deleting message: %s", err)
			}
		}
	}
	date := time.Now()
	id := generateID(date)
	return &Message{mailbox: mb, Fid: id, Fdate: date, writable: true}, nil
}

// Mailbox returns the name of the mailbox this message resides in.
func (m *Message) Mailbox() string {
	return m.mailbox.name
}

// ID gets the ID of the Message
func (m *Message) ID() string {
	return m.Fid
}

// Date returns the date/time this Message was received by Inbucket
func (m *Message) Date() time.Time {
	return m.Fdate
}

// From returns the value of the Message From header
func (m *Message) From() *mail.Address {
	return m.Ffrom
}

// To returns the value of the Message To header
func (m *Message) To() []*mail.Address {
	return m.Fto
}

// Subject returns the value of the Message Subject header
func (m *Message) Subject() string {
	return m.Fsubject
}

// String returns a string in the form: "Subject()" from From()
func (m *Message) String() string {
	return fmt.Sprintf("\"%v\" from %v", m.Fsubject, m.Ffrom)
}

// Size returns the size of the Message on disk in bytes
func (m *Message) Size() int64 {
	return m.Fsize
}

func (m *Message) rawPath() string {
	return filepath.Join(m.mailbox.path, m.Fid+".raw")
}

// ReadHeader opens the .raw portion of a Message and returns a standard Go mail.Message object
func (m *Message) ReadHeader() (msg *mail.Message, err error) {
	file, err := os.Open(m.rawPath())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Errorf("Failed to close %q: %v", m.rawPath(), err)
		}
	}()

	reader := bufio.NewReader(file)
	return mail.ReadMessage(reader)
}

// ReadBody opens the .raw portion of a Message and returns a MIMEBody object
func (m *Message) ReadBody() (body *enmime.Envelope, err error) {
	file, err := os.Open(m.rawPath())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Errorf("Failed to close %q: %v", m.rawPath(), err)
		}
	}()

	reader := bufio.NewReader(file)
	mime, err := enmime.ReadEnvelope(reader)
	if err != nil {
		return nil, err
	}
	return mime, nil
}

// RawReader opens the .raw portion of a Message as an io.ReadCloser
func (m *Message) RawReader() (reader io.ReadCloser, err error) {
	file, err := os.Open(m.rawPath())
	if err != nil {
		return nil, err
	}
	return file, nil
}

// ReadRaw opens the .raw portion of a Message and returns it as a string
func (m *Message) ReadRaw() (raw *string, err error) {
	reader, err := m.RawReader()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			log.Errorf("Failed to close %q: %v", m.rawPath(), err)
		}
	}()

	bodyBytes, err := ioutil.ReadAll(bufio.NewReader(reader))
	if err != nil {
		return nil, err
	}
	bodyString := string(bodyBytes)
	return &bodyString, nil
}

// Append data to a newly opened Message, this will fail on a pre-existing Message and
// after Close() is called.
func (m *Message) Append(data []byte) error {
	// Prevent Appending to a pre-existing Message
	if !m.writable {
		return storage.ErrNotWritable
	}
	// Open file for writing if we haven't yet
	if m.writer == nil {
		// Ensure mailbox directory exists
		if err := m.mailbox.createDir(); err != nil {
			return err
		}
		file, err := os.Create(m.rawPath())
		if err != nil {
			// Set writable false just in case something calls me a million times
			m.writable = false
			return err
		}
		m.writerFile = file
		m.writer = bufio.NewWriter(file)
	}
	_, err := m.writer.Write(data)
	m.Fsize += int64(len(data))
	return err
}

// Close this Message for writing - no more data may be Appended.  Close() will also
// trigger the creation of the .gob file.
func (m *Message) Close() error {
	// nil out the writer fields so they can't be used
	writer := m.writer
	writerFile := m.writerFile
	m.writer = nil
	m.writerFile = nil

	if writer != nil {
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	if writerFile != nil {
		if err := writerFile.Close(); err != nil {
			return err
		}
	}

	// Fetch headers
	body, err := m.ReadBody()
	if err != nil {
		return err
	}

	// Only public fields are stored in gob, hence starting with capital F
	// Parse From address
	if address, err := mail.ParseAddress(body.GetHeader("From")); err == nil {
		m.Ffrom = address
	} else {
		m.Ffrom = &mail.Address{Address: body.GetHeader("From")}
	}
	m.Fsubject = body.GetHeader("Subject")

	// Turn the To header into a slice
	if addresses, err := body.AddressList("To"); err == nil {
		m.Fto = addresses
	} else {
		m.Fto = []*mail.Address{{Address: body.GetHeader("To")}}
	}

	// Refresh the index before adding our message
	err = m.mailbox.readIndex()
	if err != nil {
		return err
	}

	// Made it this far without errors, add it to the index
	m.mailbox.messages = append(m.mailbox.messages, m)
	return m.mailbox.writeIndex()
}
