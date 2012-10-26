package smtpd

import (
	"github.com/stretchrcom/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var mailboxNames = []string{"abby", "bill", "christa", "donald", "evelyn"}

func TestFSAllMailboxes(t *testing.T) {
	fds := setupDataStore()
	defer teardownDataStore(fds)

	mboxes, err := fds.AllMailboxes()
	assert.Nil(t, err)
	assert.Equal(t, len(mboxes), 5)
}

// setupDataStore will build the following structure in a temporary
// directory:
//
// /tmp/inbucket?????????
// └── mail
//     ├── 53e
//     │   └── 53e11e
//     │       └── 53e11eb7b24cc39e33733a0ff06640f1b39425ea
//     │           ├── 20121024T164239-0000.gob
//     │           ├── 20121024T164239-0000.raw
//     │           ├── 20121025T164239-0000.gob
//     │           └── 20121025T164239-0000.raw
//     ├── 60c
//     │   └── 60c596
//     │       └── 60c5963a56da1425f133d28166ca4fe70dcb25f5
//     │           ├── 20121024T164239-0000.gob
//     │           ├── 20121024T164239-0000.raw
//     │           ├── 20121025T164239-0000.gob
//     │           └── 20121025T164239-0000.raw
//     ├── 88d
//     │   └── 88db92
//     │       └── 88db9292c772b38311e1778f6f6b18216443abf0
//     │           ├── 20121024T164239-0000.gob
//     │           ├── 20121024T164239-0000.raw
//     │           ├── 20121025T164239-0000.gob
//     │           └── 20121025T164239-0000.raw
//     ├── c69
//     │   └── c692d6
//     │       └── c692d6a10598e0a801576fdd4ecf3c37e45bfbc4
//     │           ├── 20121024T164239-0000.gob
//     │           ├── 20121024T164239-0000.raw
//     │           ├── 20121025T164239-0000.gob
//     │           └── 20121025T164239-0000.raw
//     └── e76
//         └── e76cef
//             └── e76ceff3c47adb10f62b1acd7109f88fbd5e9ca7
//                 ├── 20121024T164239-0000.gob
//                 ├── 20121024T164239-0000.raw
//                 ├── 20121025T164239-0000.gob
//                 └── 20121025T164239-0000.raw
func setupDataStore() *FileDataStore {
	// Build fake SMTP message for delivery
	testMsg := make([]byte, 0, 300)
	testMsg = append(testMsg, []byte("To: somebody@host\r\n")...)
	testMsg = append(testMsg, []byte("From: somebodyelse@host\r\n")...)
	testMsg = append(testMsg, []byte("Subject: test message\r\n")...)
	testMsg = append(testMsg, []byte("\r\n")...)
	testMsg = append(testMsg, []byte("Test Body\r\n")...)

	path, err := ioutil.TempDir("", "inbucket")
	if err != nil {
		panic(err)
	}
	mailPath := filepath.Join(path, "mail")
	ds := &FileDataStore{path: path, mailPath: mailPath}

	for _, name := range mailboxNames {
		mb, err := ds.MailboxFor(name)
		if err != nil {
			panic(err)
		}
		// Create day old message
		date := time.Now().Add(-24 * time.Hour)
		msg := &FileMessage{
			mailbox:  mb.(*FileMailbox),
			writable: true,
			Fdate:    date,
			Fid:      generatePrefix(date) + "-0000",
		}
		msg.Append(testMsg)
		if err = msg.Close(); err != nil {
			panic(err)
		}

		// Create current message
		date = time.Now()
		msg = &FileMessage{
			mailbox:  mb.(*FileMailbox),
			writable: true,
			Fdate:    date,
			Fid:      generatePrefix(date) + "-0000",
		}
		msg.Append(testMsg)
		if err = msg.Close(); err != nil {
			panic(err)
		}
	}
	return ds
}

func teardownDataStore(ds *FileDataStore) {
	if err := os.RemoveAll(ds.path); err != nil {
		panic(err)
	}
}
