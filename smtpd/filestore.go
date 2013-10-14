package smtpd

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/jhillyerd/go.enmime"
	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/log"
	"io"
	"io/ioutil"
	"net/mail"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Name of index file in each mailbox
const INDEX_FILE = "index.gob"

// We lock this when reading/writing an index file, this is a bottleneck because
// it's a single lock even if we have a million index files
var indexLock = new(sync.RWMutex)

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
type FileDataStore struct {
	path     string
	mailPath string
}

// NewFileDataStore creates a new DataStore object using the specified path
func NewFileDataStore(path string) DataStore {
	mailPath := filepath.Join(path, "mail")
	if _, err := os.Stat(mailPath); err != nil {
		// Mail datastore does not yet exist
		os.MkdirAll(mailPath, 0770)
	}
	return &FileDataStore{path: path, mailPath: mailPath}
}

// DefaultFileDataStore creates a new DataStore object.  It uses the inbucket.Config object to
// construct it's path.
func DefaultFileDataStore() DataStore {
	path, err := config.Config.String("datastore", "path")
	if err != nil {
		log.LogError("Error getting datastore path: %v", err)
		return nil
	}
	if path == "" {
		log.LogError("No value configured for datastore path")
		return nil
	}
	return NewFileDataStore(path)
}

// Retrieves the Mailbox object for a specified email address, if the mailbox
// does not exist, it will attempt to create it.
func (ds *FileDataStore) MailboxFor(emailAddress string) (Mailbox, error) {
	name := ParseMailboxName(emailAddress)
	dir := HashMailboxName(name)
	s1 := dir[0:3]
	s2 := dir[0:6]
	path := filepath.Join(ds.mailPath, s1, s2, dir)
	indexPath := filepath.Join(path, INDEX_FILE)

	return &FileMailbox{store: ds, name: name, dirName: dir, path: path,
		indexPath: indexPath}, nil
}

// AllMailboxes returns a slice with all Mailboxes
func (ds *FileDataStore) AllMailboxes() ([]Mailbox, error) {
	mailboxes := make([]Mailbox, 0, 100)
	infos1, err := ioutil.ReadDir(ds.mailPath)
	if err != nil {
		return nil, err
	}
	// Loop over level 1 directories
	for _, inf1 := range infos1 {
		if inf1.IsDir() {
			l1 := inf1.Name()
			infos2, err := ioutil.ReadDir(filepath.Join(ds.mailPath, l1))
			if err != nil {
				return nil, err
			}
			// Loop over level 2 directories
			for _, inf2 := range infos2 {
				if inf2.IsDir() {
					l2 := inf2.Name()
					infos3, err := ioutil.ReadDir(filepath.Join(ds.mailPath, l1, l2))
					if err != nil {
						return nil, err
					}
					// Loop over mailboxes
					for _, inf3 := range infos3 {
						if inf3.IsDir() {
							mbdir := inf3.Name()
							mbpath := filepath.Join(ds.mailPath, l1, l2, mbdir)
							idx := filepath.Join(mbpath, INDEX_FILE)
							mb := &FileMailbox{store: ds, dirName: mbdir, path: mbpath,
								indexPath: idx}
							mailboxes = append(mailboxes, mb)
						}
					}
				}
			}
		}
	}

	return mailboxes, nil
}

// A Mailbox manages the mail for a specific user and correlates to a particular
// directory on disk.
type FileMailbox struct {
	store       *FileDataStore
	name        string
	dirName     string
	path        string
	indexLoaded bool
	indexPath   string
	messages    []*FileMessage
}

func (mb *FileMailbox) String() string {
	return mb.name + "[" + mb.dirName + "]"
}

// GetMessages scans the mailbox directory for .gob files and decodes them into
// a slice of Message objects.
func (mb *FileMailbox) GetMessages() ([]Message, error) {
	if !mb.indexLoaded {
		if err := mb.readIndex(); err != nil {
			return nil, err
		}
	}

	messages := make([]Message, len(mb.messages))
	for i, m := range mb.messages {
		messages[i] = m
	}
	return messages, nil
}

// GetMessage decodes a single message by Id and returns a Message object
func (mb *FileMailbox) GetMessage(id string) (Message, error) {
	if !mb.indexLoaded {
		if err := mb.readIndex(); err != nil {
			return nil, err
		}
	}

	for _, m := range mb.messages {
		if m.Fid == id {
			return m, nil
		}
	}

	return nil, fmt.Errorf("Message %s not in index", id)
}

// readIndex loads the mailbox index data from disk
func (mb *FileMailbox) readIndex() error {
	// Clear message slice, open index
	mb.messages = mb.messages[:0]
	// Lock for reading
	indexLock.RLock()
	defer indexLock.RUnlock()
	// Check if index exists
	if _, err := os.Stat(mb.indexPath); err != nil {
		// Does not exist, but that's not an error in our world
		log.LogTrace("Index %v does not exist (yet)", mb.indexPath)
		mb.indexLoaded = true
		return nil
	}
	file, err := os.Open(mb.indexPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decode gob data
	dec := gob.NewDecoder(bufio.NewReader(file))
	for {
		// TODO Detect EOF
		msg := new(FileMessage)
		if err = dec.Decode(msg); err != nil {
			if err == io.EOF {
				// It's OK to get an EOF here
				break
			}
			return fmt.Errorf("While decoding message: %v", err)
		}
		msg.mailbox = mb
		log.LogTrace("Found: %v", msg)
		mb.messages = append(mb.messages, msg)
	}

	mb.indexLoaded = true
	return nil
}

// createDir checks for the presence of the path for this mailbox, creates it if needed
func (mb *FileMailbox) createDir() error {
	if _, err := os.Stat(mb.path); err != nil {
		if err := os.MkdirAll(mb.path, 0770); err != nil {
			log.LogError("Failed to create directory %v, %v", mb.path, err)
			return err
		}
	}
	return nil
}

// writeIndex overwrites the index on disk with the current mailbox data
func (mb *FileMailbox) writeIndex() error {
	// Lock for writing
	indexLock.Lock()
	defer indexLock.Unlock()
	if len(mb.messages) > 0 {
		// Ensure mailbox directory exists
		if err := mb.createDir(); err != nil {
			return err
		}
		// Open index for writing
		file, err := os.Create(mb.indexPath)
		if err != nil {
			return err
		}
		defer file.Close()
		writer := bufio.NewWriter(file)

		// Write each message and then flush
		enc := gob.NewEncoder(writer)
		for _, m := range mb.messages {
			err = enc.Encode(m)
			if err != nil {
				return err
			}
		}
		writer.Flush()
	} else {
		// No messages, delete index+maildir
		log.LogTrace("Removing mailbox %v", mb.path)
		return os.RemoveAll(mb.path)
	}

	return nil
}

// Message contains a little bit of data about a particular email message, and
// methods to retrieve the rest of it from disk.
type FileMessage struct {
	mailbox *FileMailbox
	// Stored in GOB
	Fid      string
	Fdate    time.Time
	Ffrom    string
	Fsubject string
	Fsize    int64
	// These are for creating new messages only
	writable   bool
	writerFile *os.File
	writer     *bufio.Writer
}

// NewMessage creates a new Message object and sets the Date and Id fields.
func (mb *FileMailbox) NewMessage() Message {
	date := time.Now()
	id := generateId(date)

	return &FileMessage{mailbox: mb, Fid: id, Fdate: date, writable: true}
}

func (m *FileMessage) Id() string {
	return m.Fid
}

func (m *FileMessage) Date() time.Time {
	return m.Fdate
}

func (m *FileMessage) From() string {
	return m.Ffrom
}

func (m *FileMessage) Subject() string {
	return m.Fsubject
}

func (m *FileMessage) String() string {
	return fmt.Sprintf("\"%v\" from %v", m.Fsubject, m.Ffrom)
}

func (m *FileMessage) Size() int64 {
	return m.Fsize
}

func (m *FileMessage) rawPath() string {
	return filepath.Join(m.mailbox.path, m.Fid+".raw")
}

// ReadHeader opens the .raw portion of a Message and returns a standard Go mail.Message object
func (m *FileMessage) ReadHeader() (msg *mail.Message, err error) {
	file, err := os.Open(m.rawPath())
	defer file.Close()
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(file)
	msg, err = mail.ReadMessage(reader)
	return msg, err
}

// ReadBody opens the .raw portion of a Message and returns a MIMEBody object
func (m *FileMessage) ReadBody() (body *enmime.MIMEBody, err error) {
	file, err := os.Open(m.rawPath())
	defer file.Close()
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(file)
	msg, err := mail.ReadMessage(reader)
	if err != nil {
		return nil, err
	}
	mime, err := enmime.ParseMIMEBody(msg)
	if err != nil {
		return nil, err
	}
	return mime, err
}

// RawReader opens the .raw portion of a Message as an io.ReadCloser
func (m *FileMessage) RawReader() (reader io.ReadCloser, err error) {
	file, err := os.Open(m.rawPath())
	if err != nil {
		return nil, err
	}
	return file, nil
}

// ReadRaw opens the .raw portion of a Message and returns it as a string
func (m *FileMessage) ReadRaw() (raw *string, err error) {
	reader, err := m.RawReader()
	defer reader.Close()
	if err != nil {
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(bufio.NewReader(reader))
	if err != nil {
		return nil, err
	}
	bodyString := string(bodyBytes)
	return &bodyString, nil
}

// Append data to a newly opened Message, this will fail on a pre-existing Message and
// after Close() is called.
func (m *FileMessage) Append(data []byte) error {
	// Prevent Appending to a pre-existing Message
	if !m.writable {
		return ErrNotWritable
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
func (m *FileMessage) Close() error {
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

	// Only public fields are stored in gob
	m.Ffrom = body.GetHeader("From")
	m.Fsubject = body.GetHeader("Subject")

	// Refresh the index before adding our message
	err = m.mailbox.readIndex()
	if err != nil {
		return err
	}

	// Made it this far without errors, add it to the index
	m.mailbox.messages = append(m.mailbox.messages, m)
	return m.mailbox.writeIndex()
}

// Delete this Message from disk by removing both the gob and raw files
func (m *FileMessage) Delete() error {
	messages := m.mailbox.messages
	for i, mm := range messages {
		if m == mm {
			// Slice around message we are deleting
			m.mailbox.messages = append(messages[:i], messages[i+1:]...)
			break
		}
	}
	m.mailbox.writeIndex()

	if len(m.mailbox.messages) == 0 {
		// This was the last message, writeIndex() has removed the entire
		// directory
		return nil
	}

	// There are still messages in the index
	log.LogTrace("Deleting %v", m.rawPath())
	return os.Remove(m.rawPath())
}

// generatePrefix converts a Time object into the ISO style format we use
// as a prefix for message files.  Note:  It is used directly by unit
// tests.
func generatePrefix(date time.Time) string {
	return date.Format("20060102T150405")
}

// generateId adds a 4-digit unique number onto the end of the string
// returned by generatePrefix()
func generateId(date time.Time) string {
	return generatePrefix(date) + "-" + fmt.Sprintf("%04d", <-countChannel)
}
