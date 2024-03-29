package main

import (
	"github.com/paultomas/qrzlogger/qrz"
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
var dbFile = flag.String("d", "~/.qrzlogger", "backlog file")
var offline = flag.Bool("offline", false, "Run in offline mode")
var forward = flag.String("f", "localhost:2238", "forward address")


func send(backlog Backlog, client *qrz.Client, ch <-chan string, offline bool) {
	for {
		adif := <-ch
		if offline {
			continue
		}
		err := client.Upload(adif)
		if err != nil && err != qrz.ErrAlreadyExists {
			log.Printf("ERROR: uploading the following ADIF entry. It will remain in the backlog, and will be uploaded the next time this program is started.\n%s\n", adif)
			log.Printf("ERROR: %s\n", err)			
			backlog.Add(adif)
			err = backlog.Save()
			if err != nil {
				log.Printf("ERROR: log entry \n%s\ncould not be removed from backlog - it may be uploaded more than once as a result.\n", adif)
				log.Printf("ERROR: %s\n", err)
			}
			
		} 
	}
}

func processBacklog(backlog Backlog, qrzClient *qrz.Client ) error {
	
	entries := backlog.Entries()

	if len(entries) < 1 {
		return nil
	}

	log.Printf("Entries found in backlog: %d.\n", len(entries))

	for _, adif := range entries {
		err := qrzClient.Upload(adif)
		if err !=nil && err != qrz.ErrAlreadyExists {
			log.Printf("ERROR: %v\n", err) 
			return err
		}
		if err == qrz.ErrAlreadyExists {
	        	log.Printf("Entry \n%s\n already exists, removing from backlog.\n", adif)
		} else {
        		log.Printf("Successfully uploaded entry :\n%s\n. Removing from backlog.\n", adif)
		}
		backlog.Remove(adif)
	}
	return nil
}

func listen(con *net.UDPConn, c chan<- string, forwardAddr string) {
	var clientAddr *net.UDPAddr
	var err error
	if forwardAddr != "" {
		clientAddr, err = net.ResolveUDPAddr("udp", forwardAddr)
		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())
		}
	}
	p := make([]byte, 2048)
	for {
		n, _, err := con.ReadFromUDP(p)

		if err != nil {
			log.Printf("ERROR: %v\n", err)
			continue
		}

		if n < 1 {
			continue
		}

		if clientAddr != nil {
			_, err = con.WriteToUDP(p, clientAddr)
			if err != nil {
				log.Printf("ERROR: %s\n", err.Error())
			}
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

	qrzClient := qrz.NewClient(key)
	backlog, err := NewBacklog(*dbFile)

	if err != nil {
		log.Fatal(err.Error())
	}

	err = backlog.Load()

	if err != nil {
		log.Fatal(err.Error())
	}

	if !*offline {
		err = processBacklog(backlog, qrzClient)
		if err != nil {
			log.Printf("ERROR: could not process backlog at this time %v\n", err)			
		}
		err = backlog.Save()
		if err != nil {
			log.Printf("ERROR: %s\n", err.Error())
		}
		
	} else {
		log.Printf("Running in offline mode\n")
	}

	
	inCh := make(chan string)

	go send(backlog, qrzClient, inCh, *offline)

	log.Printf("Listening on %s:%d\n", *ip, *port)
	if *forward != "" {
		log.Printf("Forwarding to %s\n", *forward)
	} else {
		log.Printf("Forwarding disabled\n")
	}
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
	
	go listen(ser, inCh, *forward)

	wg.Wait()
}
