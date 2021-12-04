package main

import (
	"database/sql"
	"encoding/binary"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

const logBookUrl = "https://logbook.qrz.com/api"

var key string
var port = flag.Int("p", 2237, "port")
var ip = flag.String("h", "0.0.0.0", "host ip")
//var key = flag.String("k", "", "API key")
var dbFile = flag.String("d", "~/.qrzlogger.sqlite3", "Database file")

func upload(adif string) error {
	form := url.Values{}
	form.Set("ACTION", "INSERT")
	form.Set("KEY", key)
	form.Set("ADIF", adif)
	client := &http.Client{}
	r, err := http.NewRequest("POST", logBookUrl, strings.NewReader(form.Encode()))

	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return err
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(form.Encode())))

	res, err := client.Do(r)
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

func addPending(adif string, db *sql.DB) error {
	stmt, err := db.Prepare("INSERT INTO entries(adif) values(?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(adif)

	return err
}

func send(db *sql.DB, ch <-chan string) {
	for {
		adif := <-ch
		if upload(adif) != nil {
			log.Printf("ERROR uploading the following ADIF entry. It will be stored for now, and then uploaded the next time this program is started.\n%s\n", adif)
			err := addPending(adif, db)
			if err != nil {
				log.Println(err.Error())
				log.Printf("Failed to capture ADIF entry. It is printed here so that you can enter it manually:\n%s\n", adif)
			}
		}
	}

}

func createTable(db *sql.DB) error {
	createTableSQL := `CREATE TABLE IF NOT EXISTS entries ( "adif" TEXT);`

	statement, err := db.Prepare(createTableSQL)
	if err != nil {
		return err
	}
	statement.Exec()
	return nil
}
func countPending(db *sql.DB) (int, error) {
	rows, err := db.Query("SELECT COUNT(*) FROM entries")
	if err != nil {
		return 0, err
	}
	var count int
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&count)
		return count, nil
	}
	return 0, errors.New("Failed to retrieve backlog count")
}

func uploadNextPending(db *sql.DB) error {
	rows, err := db.Query("SELECT adif FROM entries LIMIT 1")
	var adif string

	if !rows.Next() {
		return nil
	}
	err = rows.Scan(&adif)
	if err != nil {
		return err
	}
	rows.Close()
	err = upload(adif)
	if err != nil {
		return err
	}

	stmt, err := db.Prepare("DELETE FROM entries WHERE adif=?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(adif)
	if err != nil {
		log.Printf("ERROR: log entry \n%s\ncould not be deleted from backlog - it may be uploaded more than once as a result", adif)
		return err
	}
	return nil
}

func uploadPending(db *sql.DB) error {
	count, err := countPending(db)
	if err != nil {
		return err
	}
	if count < 1 {
		return nil
	}
	log.Printf("Number of entries in backlog: %d. Attempting to upload to your QRZ logbook.\n", count)
	for count > 0 {
		err = uploadNextPending(db)
		if err != nil {
			return err
		}
		count, err = countPending(db)
		if err != nil {
			return err
		}
	}
	return nil
}

func create(p string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}
	return os.Create(p)
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
	usr, _ := user.Current()
	homeDir := usr.HomeDir

	theDbFile := *dbFile
	if strings.HasPrefix(*dbFile, "~/") {
		theDbFile = filepath.Join(homeDir, (*dbFile)[2:])
	}
	if _, err := os.Stat(theDbFile); err != nil {
		file, err := create(theDbFile)
		if err != nil {
			log.Printf("ERROR: Failed to create file %s\n", theDbFile)
			log.Fatal(err.Error())
		}
		file.Close()
	}

	db, err := sql.Open("sqlite3", theDbFile)

	if err != nil {
		log.Fatal(err.Error())
	}

	defer db.Close()

	err = createTable(db)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = uploadPending(db)

	if err != nil {
		log.Printf("Unable to upload pending entries at this time.\n")
	}

	log.Printf("Reading from %s:%d\n", *ip, *port)

	ch := make(chan string)

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
	go listen(ser, ch)
	go send(db, ch)
	wg.Wait()
}
