package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

const LOGBOOK_URL = "https://logbook.qrz.com/api"

var key string
var port = flag.Int("p", 2237, "port")
var ip = flag.String("h", "0.0.0.0", "host ip")
var dbFile = flag.String("d", "~/.qrzlogger.sqlite3", "Database file")
var offline = flag.Bool("offline", false, "Run in offline mode")

func upload(adif string) error {
	form := url.Values{}
	form.Set("ACTION", "INSERT")
	form.Set("KEY", key)
	form.Set("ADIF", adif)
	client := &http.Client{}
	r, err := http.NewRequest("POST", LOGBOOK_URL, strings.NewReader(form.Encode()))

	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return err
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(form.Encode())))

	res, err := client.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return err
	}
	if !strings.Contains(string(body), "LOGID") {
		return errors.New(string(body))
	}
	log.Printf("Logged:\n%s\n", adif)
	log.Println(string(body))
	return nil
}

func addPending(inChan <-chan string, backlog Backlog, sendCh chan<- string) {
	for {
		adif := <-inChan
		err := backlog.Store(adif)
		if err != nil {
			log.Printf("ERROR storing entry in backlog: %v. \nEntry: \n%s", err, adif)
		}
		sendCh <- adif
	}
}

func send(backlog Backlog, ch <-chan string, offline bool) {
	for {
		adif := <-ch
		if offline {
			continue
		}
		if upload(adif) != nil {
			log.Printf("ERROR uploading the following ADIF entry. It will remain in the backlog, and will be uploaded the next time this program is started.\n%s\n", adif)
		} else {
			err := backlog.Remove(adif)
			if err != nil {
				log.Printf("ERROR: log entry \n%s\ncould not be removed from backlog - it may be uploaded more than once as a result", adif)

			}

		}
	}
}

func primeUpload(backlog Backlog, c chan<- string) error {
	entries, err := backlog.Fetch()
	if err != nil {
		return err
	}
	if len(entries) < 1 {
		return nil
	}

	log.Printf("Entries found in backlog: %d.\n", len(entries))
	for _, adif := range entries {
		c <- adif
	}
	return nil
}

func listen(con *net.UDPConn, c chan<- string) {

	p := make([]byte, 2048)
	for {
		n, _, err := con.ReadFromUDP(p)

		if err != nil {
			log.Printf("ERROR: %v", err)
			continue
		}

		if n < 1 {
			continue
		}

		magic := binary.BigEndian.Uint32(p)
		if magic != 0xadbccbda {
			log.Printf("ERROR: Unknown magic number: %x\n", magic)
			continue
		}
		var offset uint32 = 8
		payloadType := binary.BigEndian.Uint32(p[offset:])
		if payloadType == 12 {
			offset += 4
			idlen := binary.BigEndian.Uint32(p[offset:])
			offset += 4
			offset += idlen
			adifLen := binary.BigEndian.Uint32(p[offset:])
			offset += 4
			adif := string(p[offset : offset+adifLen])
			c <- adif
		}
	}
}

func main() {

	flag.Parse()
	key = os.Getenv("QRZ_KEY")
	if len(key) < 1 {
		log.Fatal("API key must be provided via the QRZ_KEY environment variable")
	}

	backlog, err := newBacklogDb(*dbFile)

	if err != nil {
		log.Fatal(err.Error())
	}

	defer backlog.Close()

	sendCh := make(chan string)

	if *offline {
		log.Printf("Running in offline mode\n")
	}
	go send(backlog, sendCh, *offline)

	if !*offline {
		err = primeUpload(backlog, sendCh)
	}

	if err != nil {
		log.Printf("ERROR: could not process backlog at this time %v\n", err)
		return
	}

	pendingCh := make(chan string)

	go addPending(pendingCh, backlog, sendCh)

	log.Printf("Reading from %s:%d\n", *ip, *port)

	addr := net.UDPAddr{
		Port: *port,
		IP:   net.ParseIP(*ip),
	}
	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go listen(ser, pendingCh)

	wg.Wait()
}
