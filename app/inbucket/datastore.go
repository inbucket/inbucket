package inbucket

import (
	"bufio"
	"fmt"
	"github.com/robfig/revel"
	"os"
	"path/filepath"
	"errors"
	"time"
)

var ErrNotWritable = errors.New("MailObject not writable")

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

type DataStore struct {
	path     string
	mailPath string
}

func NewDataStore() *DataStore {
	path, found := rev.Config.String("datastore.path")
	if found {
		mailPath := filepath.Join(path, "mail")
		return &DataStore{path: path, mailPath: mailPath}
	}
	rev.ERROR.Printf("No value configured for datastore.path")
	return nil
}

type MailObject struct {
	store      *DataStore
	mailbox    string
	rawPath    string
	gobPath    string
	writable   bool
	writerFile *os.File
	writer     *bufio.Writer
}

func (ds *DataStore) NewMailObject(emailAddress string) *MailObject {
	mailbox := ParseMailboxName(emailAddress)
	maildir := HashMailboxName(mailbox)
	fileBase := time.Now().Format("20060102T150405") + "-" + fmt.Sprintf("%04d", <-countChannel)
	boxPath := filepath.Join(ds.mailPath, maildir)
	if err := os.MkdirAll(boxPath, 0770); err != nil {
		rev.ERROR.Printf("Failed to create directory %v, %v", boxPath, err)
		return nil
	}
	pathBase := filepath.Join(boxPath, fileBase)

	return &MailObject{store: ds, mailbox: mailbox, rawPath: pathBase + ".raw",
		gobPath: pathBase + ".gob", writable: true}
}

func (m *MailObject) Mailbox() string {
	return m.mailbox
}

func (m *MailObject) Append(data []byte) error {
	// Prevent Appending to a pre-existing MailObject
	if !m.writable {
		return ErrNotWritable
	}
	// Open file for writing if we haven't yet
	if m.writer == nil {
		file, err := os.Create(m.rawPath)
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

func (m *MailObject) Close() error {
	// nil out the fields so they can't be used
	writer := m.writer
	writerFile := m.writerFile
	m.writer = nil
	m.writerFile = nil

	if (writer != nil) {
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	if (writerFile != nil) {
		if err := writerFile.Close(); err != nil {
			return err
		}
	}

	return nil
}
