// MIT License
//
// Copyright (c) 2025 Marcel Joachim Kloubert (https://marcel.coffee)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/manifoldco/promptui"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	err = loadEnvIfExists(cwd)
	if err != nil {
		log.Fatalln(err)
	}

	allSettings, err := getAllSettings()
	if err != nil {
		log.Fatalln(err)
	}

	settingNames := make([]string, 0)
	for key, _ := range allSettings {
		name := allSettings[key][""]

		settingNames = append(settingNames, name)
	}

	prompt := promptui.Select{
		Label: "Select Config",
		Items: settingNames,
	}

	_, selectedName, err := prompt.Run()
	if err != nil {
		log.Fatalln(err)
	}
	if selectedName == "" {
		return
	}

	var matchingKey *string
	for key, _ := range allSettings {
		name := allSettings[key][""]
		if name != selectedName {
			continue
		}

		matchingKey = &key
		break
	}

	if matchingKey == nil {
		return
	}

	settings := allSettings[*matchingKey]

	IMAP_HOST := strings.TrimSpace(settings["IMAP_HOST"])
	IMAP_PORT := strings.TrimSpace(settings["IMAP_PORT"])
	IMAP_USER := strings.TrimSpace(settings["IMAP_USER"])
	IMAP_PASSWORD := strings.TrimSpace(settings["IMAP_PASSWORD"])

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Loading mailboxes ..."
	s.Start()

	exitWithError := func(err error) {
		s.Stop()
		log.Fatal(err)
	}

	// create connection
	c, err := client.DialTLS(fmt.Sprintf("%v:%v", IMAP_HOST, IMAP_PORT), nil)
	if err != nil {
		exitWithError(err)
	}
	defer c.Logout()

	// login
	if err := c.Login(IMAP_USER, IMAP_PASSWORD); err != nil {
		exitWithError(err)
	}

	// list mail boxes
	mailboxInfos := make(chan *imap.MailboxInfo, 50)
	done := make(chan error, 1)

	go func() {
		done <- c.List("", "*", mailboxInfos)
	}()

	if err := <-done; err != nil {
		exitWithError(err)
	}

	s.Stop()

	if len(mailboxInfos) == 0 {
		fmt.Println("No mailboxes found")
		return
	}

	var mailboxNames []string
	for m := range mailboxInfos {
		mailboxNames = append(mailboxNames, m.Name)
	}

	sort.Slice(mailboxNames, func(x, y int) bool {
		return strings.ToLower(strings.TrimSpace(mailboxNames[x])) < strings.TrimSpace(strings.ToLower(mailboxNames[y]))
	})

	prompt = promptui.Select{
		Label: "Select Mailbox",
		Items: mailboxNames,
	}

	_, selectedMailbox, err := prompt.Run()
	if err != nil {
		log.Fatalln(err)
	}
	if selectedMailbox == "" {
		return
	}

	IMAP_FOLDER := selectedMailbox

	prompt = promptui.Select{
		Label: "Do you like to remove messages on server after download?",
		Items: []string{"Yes", "No", "Cancel"},
	}
	_, shouldRemoveMessagesOnServerSelection, err := prompt.Run()
	if err != nil {
		log.Fatalln(err)
	}
	if shouldRemoveMessagesOnServerSelection == "Cancel" || shouldRemoveMessagesOnServerSelection == "" {
		return
	}

	shouldRemoveMessagesOnServer := shouldRemoveMessagesOnServerSelection == "Yes"

	backupDir := filepath.Join(cwd, fmt.Sprintf("backups/%s/%s", sanitize(IMAP_USER), sanitize(IMAP_FOLDER)))
	err = os.MkdirAll(backupDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Loading messages ..."
	s.Start()
	defer s.Stop()

	// open INBOX
	mbox, err := c.Select(IMAP_FOLDER, false)
	if err != nil {
		log.Fatal(err)
	}

	l := func(msg any) {
		s.Suffix = fmt.Sprintf(" %v", msg)
	}

	if mbox.Messages == 0 {
		fmt.Println("No messages available")
		return
	}

	l(fmt.Sprintf("Selecting %v messages ...", mbox.Messages))

	// define message range (in this case all)
	seqset := new(imap.SeqSet)
	seqset.AddRange(1, mbox.Messages)

	// body
	section := &imap.BodySectionName{}

	// header
	items := []imap.FetchItem{imap.FetchEnvelope, section.FetchItem()}

	messages := make(chan *imap.Message, 10)
	done = make(chan error, 1)

	go func() {
		done <- c.Fetch(seqset, items, messages)
	}()

	var toDelete imap.SeqSet

	for msg := range messages {
		l(fmt.Sprintf("Handling message %v ...", msg.SeqNum))

		func() {
			if msg == nil {
				return
			}

			subject := "no-subject"
			from := "unknown"
			timestamp := "unknown-date"

			rawData := msg.GetBody(section)

			if msg.Envelope != nil {
				if msg.Envelope.Subject != "" {
					subject = msg.Envelope.Subject
				}
				if len(msg.Envelope.From) > 0 {
					fromAddr := msg.Envelope.From[0]
					from = fromAddr.MailboxName + "@" + fromAddr.HostName
				}
				if !msg.Envelope.Date.IsZero() {
					timestamp = msg.Envelope.Date.Format("20060102150405")
				}
			}

			// parts for final filename
			safeSubject := sanitize(subject)
			safeFrom := sanitize(from)
			safeTimestamp := sanitize(timestamp)

			// create full path
			filename := fmt.Sprintf("email-%d_%s_%s_%s.eml", msg.SeqNum, safeTimestamp, safeFrom, safeSubject)
			filepath := filepath.Join(backupDir, filename)

			l(fmt.Sprintf("Saving message to %v ...", filepath))

			// write to output ...
			file, err := os.Create(filepath)
			if err != nil {
				log.Printf("Could not create file %s: %v%v", filename, err, fmt.Sprintln())
				return
			}
			defer file.Close()

			writer := bufio.NewWriter(file)
			if rawData != nil {
				_, err = io.Copy(writer, rawData)
			}
			writer.Flush()

			if err != nil {
				log.Printf("Could not write to %s: %v%v", filename, err, fmt.Sprintln())
				return
			}

			gzPath := filepath + ".gz"
			l(fmt.Sprintf("Zipping message to %v ...", gzPath))

			// compress with GZIP
			if err := compressGZIP(filepath, gzPath); err != nil {
				log.Printf("Could not compress %v: %v%v", filepath, err, fmt.Sprintln())
			} else {
				err := os.Remove(filepath)
				if err != nil {
					log.Printf("WARN: Could not delete %v: %v%v", filepath, err, fmt.Sprintln())
				}
			}

			// add to list of messages that should be delted
			toDelete.AddNum(msg.SeqNum)
		}()
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	if shouldRemoveMessagesOnServer {
		// remove from server

		l(fmt.Sprintf("Deleting %v message ...", len(toDelete.Set)))

		if err := c.Store(&toDelete, "+FLAGS.SILENT", []interface{}{imap.DeletedFlag}, nil); err != nil {
			log.Printf("Could not mark messages as DELETED: %v%v", err, fmt.Sprintln())
		}

		if err := c.Expunge(nil); err != nil {
			log.Printf("Expunge error: %v%v", err, fmt.Sprintln())
		}
	}

	l("Done")
}
