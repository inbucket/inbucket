package file

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/inbucket/inbucket/v3/pkg/stringutil"
	"github.com/rs/zerolog/log"
)

// Name of index file in each mailbox
const indexFileName = "index.gob"

var (
	// countChannel is filled with a sequential numbers (0000..9999), which are
	// used by generateID() to generate unique message IDs.  It's global
	// because we only want one regardless of the number of DataStore objects.
	countChannel = make(chan int, 10)
)

func init() {
	// Start generator
	go countGenerator(countChannel)
}

// Populates the channel with numbers.
func countGenerator(c chan int) {
	for i := 0; true; i = (i + 1) % 10000 {
		c <- i
	}
}

// Store implements DataStore aand is the root of the mail storage
// hiearchy.  It provides access to Mailbox objects.
type Store struct {
	hashLock      storage.HashLock
	path          string
	mailPath      string
	messageCap    int
	bufReaderPool sync.Pool
	extHost       *extension.Host
}

// New creates a new DataStore object using the specified path.
func New(cfg config.Storage, extHost *extension.Host) (storage.Store, error) {
	path := cfg.Params["path"]
	if path == "" {
		return nil, errors.New("'path' parameter not specified")
	}

	mailPath := getMailPath(path)
	if _, err := os.Stat(mailPath); err != nil {
		// Mail datastore does not yet exist, create it.
		if err = os.MkdirAll(mailPath, 0770); err != nil {
			log.Error().Str("module", "storage").Str("path", mailPath).Err(err).
				Msg("Error creating dir")
			return nil, err
		}
	}

	return &Store{
		path:       path,
		mailPath:   mailPath,
		messageCap: cfg.MailboxMsgCap,
		bufReaderPool: sync.Pool{
			New: func() interface{} {
				return bufio.NewReader(nil)
			},
		},
		extHost: extHost,
	}, nil
}

// AddMessage adds a message to the specified mailbox.
func (fs *Store) AddMessage(m storage.Message) (id string, err error) {
	mb := fs.mbox(m.Mailbox())
	mb.Lock()
	defer mb.Unlock()
	r, err := m.Source()
	if err != nil {
		return "", err
	}

	// Create a new message.
	fm, err := mb.newMessage()
	if err != nil {
		return "", err
	}

	// Ensure mailbox directory exists.
	if err := mb.createDir(); err != nil {
		return "", err
	}

	// Write the message content.
	file, err := os.Create(fm.rawPath())
	if err != nil {
		return "", err
	}
	w := bufio.NewWriter(file)
	size, err := io.Copy(w, r)
	if err != nil {
		// Try to remove the file.
		_ = file.Close()
		_ = os.Remove(fm.rawPath())
		return "", err
	}
	_ = r.Close()
	if err := w.Flush(); err != nil {
		// Try to remove the file.
		_ = file.Close()
		_ = os.Remove(fm.rawPath())
		return "", err
	}
	if err := file.Close(); err != nil {
		// Try to remove the file.
		_ = os.Remove(fm.rawPath())
		return "", err
	}

	// Update the index.
	fm.Fdate = m.Date()
	fm.Ffrom = m.From()
	fm.Fto = m.To()
	fm.Fsize = size
	fm.Fsubject = m.Subject()
	mb.messages = append(mb.messages, fm)
	if err := mb.writeIndex(); err != nil {
		// Try to remove the file.
		_ = os.Remove(fm.rawPath())
		return "", err
	}

	return fm.Fid, nil
}

// GetMessage returns the messages in the named mailbox, or an error.
func (fs *Store) GetMessage(mailbox, id string) (storage.Message, error) {
	mb := fs.mbox(mailbox)
	mb.RLock()
	defer mb.RUnlock()
	return mb.getMessage(id)
}

// GetMessages returns the messages in the named mailbox, or an error.
func (fs *Store) GetMessages(mailbox string) ([]storage.Message, error) {
	mb := fs.mbox(mailbox)
	mb.RLock()
	defer mb.RUnlock()
	return mb.getMessages()
}

// MarkSeen flags the message as having been read.
func (fs *Store) MarkSeen(mailbox, id string) error {
	mb := fs.mbox(mailbox)
	mb.Lock()
	defer mb.Unlock()

	if !mb.indexLoaded {
		if err := mb.readIndex(); err != nil {
			return err
		}
	}

	for _, m := range mb.messages {
		if m.Fid == id {
			if m.Fseen {
				// Already marked seen.
				return nil
			}
			m.Fseen = true
			break
		}
	}

	return mb.writeIndex()
}

// RemoveMessage deletes a message by ID from the specified mailbox.
func (fs *Store) RemoveMessage(mailbox, id string) error {
	mb := fs.mbox(mailbox)
	mb.Lock()
	defer mb.Unlock()
	return mb.removeMessage(id)
}

// PurgeMessages deletes all messages in the named mailbox, or returns an error.
func (fs *Store) PurgeMessages(mailbox string) error {
	mb := fs.mbox(mailbox)
	mb.Lock()
	defer mb.Unlock()

	// Emit delete events.
	if !mb.indexLoaded {
		if err := mb.readIndex(); err != nil {
			return err
		}
	}
	for _, m := range mb.messages {
		fs.extHost.Events.AfterMessageDeleted.Emit(message.MakeMetadata(m))
	}

	return mb.purge()
}

// VisitMailboxes accepts a function that will be called with the messages in each mailbox while it
// continues to return true.
func (fs *Store) VisitMailboxes(f func([]storage.Message) (cont bool)) error {
	names1, err := readDirNames(fs.mailPath)
	if err != nil {
		return err
	}

	// Loop over level 1 directories.
	for _, name1 := range names1 {
		names2, err := readDirNames(fs.mailPath, name1)
		if err != nil {
			return err
		}

		// Loop over level 2 directories.
		for _, name2 := range names2 {
			names3, err := readDirNames(fs.mailPath, name1, name2)
			if err != nil {
				return err
			}

			// Loop over mailboxes.
			for _, name3 := range names3 {
				mb := fs.mboxFromHash(name3)
				mb.RLock()
				msgs, err := mb.getMessages()
				mb.RUnlock()
				if err != nil {
					return err
				}
				if !f(msgs) {
					return nil
				}
			}
		}
	}
	return nil
}

// mbox returns the named mailbox.
func (fs *Store) mbox(mailbox string) *mbox {
	hash := stringutil.HashMailboxName(mailbox)
	s1 := hash[0:3]
	s2 := hash[0:6]
	path := filepath.Join(fs.mailPath, s1, s2, hash)
	indexPath := filepath.Join(path, indexFileName)

	return &mbox{
		RWMutex:   fs.hashLock.Get(hash),
		store:     fs,
		name:      mailbox,
		dirName:   hash,
		path:      path,
		indexPath: indexPath,
	}
}

// mboxFromPath constructs a mailbox based on name hash.
func (fs *Store) mboxFromHash(hash string) *mbox {
	s1 := hash[0:3]
	s2 := hash[0:6]
	path := filepath.Join(fs.mailPath, s1, s2, hash)
	indexPath := filepath.Join(path, indexFileName)

	return &mbox{
		RWMutex:   fs.hashLock.Get(hash),
		store:     fs,
		dirName:   hash,
		path:      path,
		indexPath: indexPath,
	}
}

// getPooledReader pulls a buffered reader from the fs.bufReaderPool.
func (fs *Store) getPooledReader(r io.Reader) *bufio.Reader {
	br := fs.bufReaderPool.Get().(*bufio.Reader)
	br.Reset(r)
	return br
}

// putPooledReader returns a buffered reader to the fs.bufReaderPool.
func (fs *Store) putPooledReader(br *bufio.Reader) {
	fs.bufReaderPool.Put(br)
}

// generatePrefix converts a Time object into the ISO style format we use
// as a prefix for message files.  Note:  It is used directly by unit
// tests.
func generatePrefix(date time.Time) string {
	return date.Format("20060102T150405")
}

// generateId adds a 4-digit unique number onto the end of the string
// returned by generatePrefix().
func generateID(date time.Time) string {
	return generatePrefix(date) + "-" + fmt.Sprintf("%04d", <-countChannel)
}

// getMailPath converts a filestore `path` parameter into the effective mail store path.
// Within the path, '$' is replaced with ':' to support Windows drive letters with our
// env->config map syntax.
func getMailPath(base string) string {
	path := strings.ReplaceAll(base, "$", ":")
	return filepath.Join(path, "mail")
}

// readDirNames returns a slice of filenames in the specified directory or an error.
func readDirNames(elem ...string) ([]string, error) {
	f, err := os.Open(filepath.Join(elem...))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	return f.Readdirnames(0)
}
