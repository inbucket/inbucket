package mem

import "container/list"

type msgDone struct {
	msg  *Message
	done chan struct{}
}

// maxSizeEnforcer will delete the oldest message until the entire mail store is equal to or less
// than maxSize bytes.
func (s *Store) maxSizeEnforcer(maxSize int64) {
	all := &list.List{}
	curSize := int64(0)
	for {
		select {
		case md, ok := <-s.incoming:
			if !ok {
				return
			}
			// Add message to all.
			m := md.msg
			el := all.PushBack(m)
			m.el = el
			curSize += int64(m.Size())
			for curSize > maxSize {
				// Remove oldest message.
				el := all.Front()
				all.Remove(el)
				m := el.Value.(*Message)
				if s.removeMessage(m.mailbox, m.id) != nil {
					curSize -= int64(m.Size())
				}
			}
			close(md.done)
		case md, ok := <-s.remove:
			if !ok {
				return
			}
			// Remove message from all.
			m := md.msg
			el := all.Remove(m.el)
			if el != nil {
				curSize -= int64(m.Size())
			}
			close(md.done)
		}
	}
}

// enforcerDeliver sends delivery to enforcer if configured, and waits for completion.
func (s *Store) enforcerDeliver(m *Message) {
	if s.incoming != nil {
		md := &msgDone{
			msg:  m,
			done: make(chan struct{}),
		}
		s.incoming <- md
		<-md.done
	}
}

// enforcerRemove sends removal to enforcer if configured, and waits for completion.
func (s *Store) enforcerRemove(m *Message) {
	if s.remove != nil {
		md := &msgDone{
			msg:  m,
			done: make(chan struct{}),
		}
		s.remove <- md
		<-md.done
	}
}
