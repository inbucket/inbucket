package mem

import (
	"io/ioutil"
	"sort"
	"strconv"
	"sync"

	"github.com/jhillyerd/inbucket/pkg/config"
	"github.com/jhillyerd/inbucket/pkg/storage"
)

// Store implements an in-memory message store.
type Store struct {
	sync.Mutex
	boxes map[string]*mbox
	cap   int
}

type mbox struct {
	sync.RWMutex
	name     string
	last     int
	first    int
	messages map[string]*Message
}

var _ storage.Store = &Store{}

// New returns an emtpy memory store.
func New(cfg config.Storage) (storage.Store, error) {
	return &Store{
		boxes: make(map[string]*mbox),
		cap:   cfg.MailboxMsgCap,
	}, nil
}

// AddMessage stores the message, message ID and Size will be ignored.
func (s *Store) AddMessage(message storage.Message) (id string, err error) {
	s.withMailbox(message.Mailbox(), true, func(mb *mbox) {
		r, ierr := message.Source()
		if ierr != nil {
			err = ierr
			return
		}
		source, ierr := ioutil.ReadAll(r)
		if ierr != nil {
			err = ierr
			return
		}
		// Generate message ID.
		mb.last++
		id = strconv.Itoa(mb.last)
		m := &Message{
			index:   mb.last,
			mailbox: message.Mailbox(),
			id:      id,
			from:    message.From(),
			to:      message.To(),
			date:    message.Date(),
			subject: message.Subject(),
			source:  source,
		}
		mb.messages[id] = m
		if s.cap > 0 {
			// Enforce cap.
			for len(mb.messages) > s.cap {
				delete(mb.messages, strconv.Itoa(mb.first))
				mb.first++
			}
		}
	})
	return id, err
}

// GetMessage gets a mesage.
func (s *Store) GetMessage(mailbox, id string) (m storage.Message, err error) {
	s.withMailbox(mailbox, false, func(mb *mbox) {
		m = mb.messages[id]
	})
	return m, err
}

// GetMessages gets a list of messages.
func (s *Store) GetMessages(mailbox string) (ms []storage.Message, err error) {
	s.withMailbox(mailbox, false, func(mb *mbox) {
		ms = make([]storage.Message, 0, len(mb.messages))
		for _, v := range mb.messages {
			ms = append(ms, v)
		}
		sort.Slice(ms, func(i, j int) bool {
			return ms[i].(*Message).index < ms[j].(*Message).index
		})
	})
	return ms, err
}

// PurgeMessages deletes the contents of a mailbox.
func (s *Store) PurgeMessages(mailbox string) error {
	s.withMailbox(mailbox, true, func(mb *mbox) {
		mb.messages = make(map[string]*Message)
	})
	return nil
}

// RemoveMessage deletes a single message.
func (s *Store) RemoveMessage(mailbox, id string) error {
	s.withMailbox(mailbox, true, func(mb *mbox) {
		delete(mb.messages, id)
	})
	return nil
}

// VisitMailboxes visits each mailbox in the store.
func (s *Store) VisitMailboxes(f func([]storage.Message) (cont bool)) error {
	// Lock store, get names of all mailboxes.
	s.Lock()
	boxNames := make([]string, 0, len(s.boxes))
	for k := range s.boxes {
		boxNames = append(boxNames, k)
	}
	s.Unlock()
	// Process mailboxes.
	for _, mailbox := range boxNames {
		ms, _ := s.GetMessages(mailbox)
		if !f(ms) {
			break
		}
	}
	return nil
}

// withMailbox gets or creates a mailbox, locks it, then calls f.
func (s *Store) withMailbox(mailbox string, writeLock bool, f func(mb *mbox)) {
	s.Lock()
	mb, ok := s.boxes[mailbox]
	if !ok {
		// Create mailbox
		mb = &mbox{
			name:     mailbox,
			messages: make(map[string]*Message),
		}
		s.boxes[mailbox] = mb
	}
	s.Unlock()
	if writeLock {
		mb.Lock()
	} else {
		mb.RLock()
	}
	defer func() {
		if writeLock {
			mb.Unlock()
		} else {
			mb.RUnlock()
		}
	}()
	f(mb)
}
