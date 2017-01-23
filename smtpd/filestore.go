package smtpd

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/log"
)

// Name of index file in each mailbox
const indexFileName = "index.gob"

var (
	// indexLock is locked while reading/writing an index file
	//
	// NOTE: This is a bottleneck because it's a single lock even if we have a
	// million index files
	indexLock = new(sync.RWMutex)

	// countChannel is filled with a sequential numbers (0000..9999), which are
	// used by generateID() to generate unique message IDs.  It's global
	// because we only want one regardless of the number of DataStore objects
	countChannel = make(chan int, 10)
)

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

// FileDataStore implements DataStore aand is the root of the mail storage
// hiearchy.  It provides access to Mailbox objects
type FileDataStore struct {
	path       string
	mailPath   string
	messageCap int
}

// NewFileDataStore creates a new DataStore object using the specified path
func NewFileDataStore(cfg config.DataStoreConfig) DataStore {
	path := cfg.Path
	if path == "" {
		log.Errorf("No value configured for datastore path")
		return nil
	}
	mailPath := filepath.Join(path, "mail")
	if _, err := os.Stat(mailPath); err != nil {
		// Mail datastore does not yet exist
		if err = os.MkdirAll(mailPath, 0770); err != nil {
			log.Errorf("Error creating dir %q: %v", mailPath, err)
		}
	}
	return &FileDataStore{path: path, mailPath: mailPath, messageCap: cfg.MailboxMsgCap}
}

// DefaultFileDataStore creates a new DataStore object.  It uses the inbucket.Config object to
// construct it's path.
func DefaultFileDataStore() DataStore {
	cfg := config.GetDataStoreConfig()
	return NewFileDataStore(cfg)
}

// MailboxFor retrieves the Mailbox object for a specified email address, if the mailbox
// does not exist, it will attempt to create it.
func (ds *FileDataStore) MailboxFor(emailAddress string) (Mailbox, error) {
	name, err := ParseMailboxName(emailAddress)
	if err != nil {
		return nil, err
	}
	dir := HashMailboxName(name)
	s1 := dir[0:3]
	s2 := dir[0:6]
	path := filepath.Join(ds.mailPath, s1, s2, dir)
	indexPath := filepath.Join(path, indexFileName)

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
							idx := filepath.Join(mbpath, indexFileName)
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

// FileMailbox implements Mailbox, manages the mail for a specific user and
// correlates to a particular directory on disk.
type FileMailbox struct {
	store       *FileDataStore
	name        string
	dirName     string
	path        string
	indexLoaded bool
	indexPath   string
	messages    []*FileMessage
}

func (mb *FileMailbox) Name() string {
	return mb.name
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

	return nil, ErrNotExist
}

// Purge deletes all messages in this mailbox
func (mb *FileMailbox) Purge() error {
	mb.messages = mb.messages[:0]
	return mb.writeIndex()
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
		log.Tracef("Index %v does not exist (yet)", mb.indexPath)
		mb.indexLoaded = true
		return nil
	}
	file, err := os.Open(mb.indexPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Errorf("Failed to close %q: %v", mb.indexPath, err)
		}
	}()

	// Decode gob data
	dec := gob.NewDecoder(bufio.NewReader(file))
	for {
		msg := new(FileMessage)
		if err = dec.Decode(msg); err != nil {
			if err == io.EOF {
				// It's OK to get an EOF here
				break
			}
			return fmt.Errorf("Corrupt mailbox %q: %v", mb.indexPath, err)
		}
		msg.mailbox = mb
		mb.messages = append(mb.messages, msg)
	}

	mb.indexLoaded = true
	return nil
}

// createDir checks for the presence of the path for this mailbox, creates it if needed
func (mb *FileMailbox) createDir() error {
	if _, err := os.Stat(mb.path); err != nil {
		if err := os.MkdirAll(mb.path, 0770); err != nil {
			log.Errorf("Failed to create directory %v, %v", mb.path, err)
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
		defer func() {
			if err := file.Close(); err != nil {
				log.Errorf("Failed to close %q: %v", mb.indexPath, err)
			}
		}()
		writer := bufio.NewWriter(file)

		// Write each message and then flush
		enc := gob.NewEncoder(writer)
		for _, m := range mb.messages {
			err = enc.Encode(m)
			if err != nil {
				return err
			}
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	} else {
		// No messages, delete index+maildir
		log.Tracef("Removing mailbox %v", mb.path)
		return os.RemoveAll(mb.path)
	}

	return nil
}

// generatePrefix converts a Time object into the ISO style format we use
// as a prefix for message files.  Note:  It is used directly by unit
// tests.
func generatePrefix(date time.Time) string {
	return date.Format("20060102T150405")
}

// generateId adds a 4-digit unique number onto the end of the string
// returned by generatePrefix()
func generateID(date time.Time) string {
	return generatePrefix(date) + "-" + fmt.Sprintf("%04d", <-countChannel)
}
