package client_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/inbucket/inbucket/v3/pkg/rest/client"
)

// Example demonstrates basic usage for the Inbucket REST client.
func Example() {
	// Setup a fake Inbucket server for this example.
	baseURL, teardown := exampleSetup()
	defer teardown()

	err := func() error {
		// Begin by creating a new client using the base URL of your Inbucket server, i.e.
		// `localhost:9000`.
		restClient, err := client.New(baseURL)
		if err != nil {
			return err
		}

		// Get a slice of message headers for the mailbox named `user1`.
		headers, err := restClient.ListMailbox("user1")
		if err != nil {
			return err
		}
		for _, header := range headers {
			fmt.Printf("ID: %v, Subject: %v\n", header.ID, header.Subject)
		}

		// Get the content of the first message.
		message, err := headers[0].GetMessage()
		if err != nil {
			return err
		}
		fmt.Printf("\nFrom: %v\n", message.From)
		fmt.Printf("Text body:\n%v", message.Body.Text)

		// Delete the second message.
		err = headers[1].Delete()
		if err != nil {
			return err
		}

		return nil
	}()

	if err != nil {
		log.Print(err)
	}

	// Output:
	// ID: 20180107T224128-0000, Subject: First subject
	// ID: 20180108T121212-0123, Subject: Second subject
	//
	// From: admin@inbucket.org
	// Text body:
	// This is the plain text body
}

// exampleSetup creates a fake Inbucket server to power Example() below.
func exampleSetup() (baseURL string, teardown func()) {
	router := mux.NewRouter()
	server := httptest.NewServer(router)

	// Handle ListMailbox request.
	router.HandleFunc("/api/v1/mailbox/user1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{
				"mailbox": "user1",
				"id": "20180107T224128-0000",
				"subject": "First subject"
			},
			{
				"mailbox": "user1",
				"id": "20180108T121212-0123",
				"subject": "Second subject"
			}
		]`))
	})

	// Handle GetMessage request.
	router.HandleFunc("/api/v1/mailbox/user1/20180107T224128-0000",
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{
				"mailbox": "user1",
				"id": "20180107T224128-0000",
				"from": "admin@inbucket.org",
				"subject": "First subject",
				"body": {
					"text": "This is the plain text body"
				}
			}`))
		})

	// Handle Delete request.
	router.HandleFunc("/api/v1/mailbox/user1/20180108T121212-0123",
		func(w http.ResponseWriter, r *http.Request) {
			// Nop.
		})

	return server.URL, func() {
		server.Close()
	}
}
