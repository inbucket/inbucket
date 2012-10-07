package inbucket

import "testing"

func TestParseMailboxName(t *testing.T) {
  in, out := "MailBOX", "mailbox"
  if x := ParseMailboxName(in); x != out {
    t.Errorf("ParseMailboxName(%v) = %v, want %v", in, x, out)
  }

  in, out = "MailBox@Host.Com", "mailbox"
  if x := ParseMailboxName(in); x != out {
    t.Errorf("ParseMailboxName(%v) = %v, want %v", in, x, out)
  }

  in, out = "Mail+extra@Host.Com", "mail"
  if x := ParseMailboxName(in); x != out {
    t.Errorf("ParseMailboxName(%v) = %v, want %v", in, x, out)
  }
}

func TestHashMailboxName(t *testing.T) {
  in, out := "mail", "1d6e1cf70ec6f9ab28d3ea4b27a49a77654d370e"
  if x := HashMailboxName(in); x != out {
    t.Errorf("HashMailboxName(%v) = %v, want %v", in, x, out)
  }
}

