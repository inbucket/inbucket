package inbucket

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrNotWritable = errors.New("Message not writable")

// Global because we only want one regardless of the number of DataStore objects
var countChannel = make(chan int, 10)

func init() {
	// Start generator
	go countGenerator(countChannel)
}

// Populates the channel with numbers
func countGenerator(c chan int) {
	for i := 0; true; i = (i + 1) % 10000 {
		c <- i
	}
}

// A DataStore is the root of the mail storage hiearchy.  It provides access to
// Mailbox objects
type DataStore struct {
	path     string
	mailPath string
}

// NewDataStore creates a new DataStore object.  It uses the inbucket.Config object to
// construct it's path.
func NewDataStore() *DataStore {
	path, err := Config.String("datastore", "path")
	if err != nil {
		Error("Error getting datastore path: %v", err)
		return nil
	}
	if path == "" {
		Error("No value configured for datastore path")
		return nil
	}
	mailPath := filepath.Join(path, "mail")
	return &DataStore{path: path, mailPath: mailPath}
}

// Retrieves the Mailbox object for a specified email address, if the mailbox
// does not exist, it will attempt to create it.
func (ds *DataStore) MailboxFor(emailAddress string) (*Mailbox, error) {
	name := ParseMailboxName(emailAddress)
	dir := HashMailboxName(name)
	path := filepath.Join(ds.mailPath, dir)
	if err := os.MkdirAll(path, 0770); err != nil {
		Error("Failed to create directory %v, %v", path, err)
		return nil, err
	}
	return &Mailbox{store: ds, name: name, dirName: dir, path: path}, nil
}

// A Mailbox manages the mail for a specific user and correlates to a particular
// directory on disk.
type Mailbox struct {
	store   *DataStore
	name    string
	dirName string
	path    string
}

func (mb *Mailbox) String() string {
	return mb.name + "[" + mb.dirName + "]"
}

// GetMessages scans the mailbox directory for .gob files and decodes them into
// a slice of Message objects.
func (mb *Mailbox) GetMessages() ([]*Message, error) {
	files, err := ioutil.ReadDir(mb.path)
	if err != nil {
		return nil, err
	}
	Trace("Scanning %v files for %v", len(files), mb)

	messages := make([]*Message, 0, len(files))
	for _, f := range files {
		if (!f.IsDir()) && strings.HasSuffix(strings.ToLower(f.Name()), ".gob") {
			// We have a gob file
			file, err := os.Open(filepath.Join(mb.path, f.Name()))
			if err != nil {
				return nil, err
			}
			dec := gob.NewDecoder(bufio.NewReader(file))
			msg := new(Message)
			if err = dec.Decode(msg); err != nil {
				return nil, err
			}
			file.Close()
			msg.mailbox = mb
			Trace("Found: %v", msg)
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

// GetMessage decodes a single message by Id and returns a Message object
func (mb *Mailbox) GetMessage(id string) (*Message, error) {
	file, err := os.Open(filepath.Join(mb.path, id+".gob"))
	if err != nil {
		return nil, err
	}

	dec := gob.NewDecoder(bufio.NewReader(file))
	msg := new(Message)
	if err = dec.Decode(msg); err != nil {
		return nil, err
	}
	file.Close()
	msg.mailbox = mb
	Trace("Found: %v", msg)

	return msg, nil
}

// Message contains a little bit of data about a particular email message, and
// methods to retrieve the rest of it from disk.
type Message struct {
	mailbox *Mailbox
	Id      string
	Date    time.Time
	From    string
	Subject string
	// These are for creating new messages only
	writable   bool
	writerFile *os.File
	writer     *bufio.Writer
}

// NewMessage creates a new Message object and sets the Date and Id fields.
func (mb *Mailbox) NewMessage() *Message {
	date := time.Now()
	id := date.Format("20060102T150405") + "-" + fmt.Sprintf("%04d", <-countChannel)

	return &Message{mailbox: mb, Id: id, Date: date, writable: true}
}

func (m *Message) String() string {
	return fmt.Sprintf("\"%v\" from %v", m.Subject, m.From)
}

func (m *Message) gobPath() string {
	return filepath.Join(m.mailbox.path, m.Id+".gob")
}

func (m *Message) rawPath() string {
	return filepath.Join(m.mailbox.path, m.Id+".raw")
}

// ReadHeader opens the .raw portion of a Message and returns a standard Go mail.Message object
func (m *Message) ReadHeader() (msg *mail.Message, err error) {
	file, err := os.Open(m.rawPath())
	defer file.Close()
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(file)
	msg, err = mail.ReadMessage(reader)
	return msg, err
}

// ReadBody opens the .raw portion of a Message and returns a MIMEBody object, along
// with a free mail.Message containing the Headers, since we had to make one of those
// anyway.
func (m *Message) ReadBody() (msg *mail.Message, body *MIMEBody, err error) {
	file, err := os.Open(m.rawPath())
	defer file.Close()
	if err != nil {
		return nil, nil, err
	}
	reader := bufio.NewReader(file)
	msg, err = mail.ReadMessage(reader)
	if err != nil {
		return nil, nil, err
	}
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		return nil, nil, err
	}
	return msg, mime, err
}

// ReadRaw opens the .raw portion of a Message and returns it as a string
func (m *Message) ReadRaw() (raw *string, err error) {
	file, err := os.Open(m.rawPath())
	defer file.Close()
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(file)
	bodyBytes, err := ioutil.ReadAll(reader)
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
		return ErrNotWritable
	}
	// Open file for writing if we haven't yet
	if m.writer == nil {
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

	return m.createGob()
}

// Delete this Message from disk by removing both the gob and raw files
func (m *Message) Delete() error {
	Trace("Deleting %v", m.gobPath())
	err := os.Remove(m.gobPath())
	if err != nil {
		return err
	}
	Trace("Deleting %v", m.rawPath())
	return os.Remove(m.rawPath())
}

// createGob reads the .raw file to grab the From and Subject header entries,
// then creates the .gob file.
func (m *Message) createGob() error {
	// Open gob for writing
	file, err := os.Create(m.gobPath())
	defer file.Close()
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(file)

	// Fetch headers
	msg, err := m.ReadHeader()
	if err != nil {
		return err
	}

	// Only public fields are stored in gob
	m.From = msg.Header.Get("From")
	m.Subject = msg.Header.Get("Subject")

	// Write & flush
	enc := gob.NewEncoder(writer)
	err = enc.Encode(m)
	if err != nil {
		return err
	}
	writer.Flush()
	return nil
}
