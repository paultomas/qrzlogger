package main

import (
	"encoding/binary"
	"flag"
	"log"
	"net"
	"os"
	"sync"
)

var key string
var port = flag.Int("p", 2237, "port")
var ip = flag.String("h", "0.0.0.0", "host ip")
var dbFile = flag.String("d", "~/.qrzlogger.sqlite3", "Database file")
var offline = flag.Bool("offline", false, "Run in offline mode")

func pushToBacklog(inChan <-chan string, backlog Backlog, sendCh chan<- string) {
	for {
		adif := <-inChan
		err := backlog.Store(adif)
		if err != nil {
			log.Printf("ERROR: storing entry in backlog: %v. \nEntry: \n%s", err, adif)
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
			log.Printf("ERROR: uploading the following ADIF entry. It will remain in the backlog, and will be uploaded the next time this program is started.\n%s\n", adif)
		} else {
			err := backlog.Remove(adif)
			if err != nil {
				log.Printf("ERROR: log entry \n%s\ncould not be removed from backlog - it may be uploaded more than once as a result", adif)

			}

		}
	}
}

func processBacklog(backlog Backlog, c chan<- string) error {
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
		err = processBacklog(backlog, sendCh)
	}

	if err != nil {
		log.Printf("ERROR: could not process backlog at this time %v\n", err)
		return
	}

	pendingCh := make(chan string)

	go pushToBacklog(pendingCh, backlog, sendCh)

	log.Printf("Listening on %s:%d\n", *ip, *port)

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
