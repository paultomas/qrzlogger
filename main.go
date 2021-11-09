package main
import (
    "flag"
    "fmt"
    "net"
    "encoding/binary"
    "net/http"
    "net/url"
    "io/ioutil"
    "strconv"
    "strings"
    "log"
    "time"
) 

const logBookUrl = "https://logbook.qrz.com/api"

var port = flag.Int("p", 2237, "port")
var ip = flag.String("h", "0.0.0.0", "host ip")
var key = flag.String("k", "", "API key")

func send(ch chan string) {
 for {
    select {
    case adif := <- ch :
    form := url.Values{}
    form.Set("ACTION", "INSERT")
    form.Set("KEY", *key)
    form.Set("ADIF", adif)
    client := &http.Client{}
    r, err := http.NewRequest("POST", logBookUrl, strings.NewReader(form.Encode()))  

    if err != nil {
        log.Printf("ERROR: %v\n", err)
	continue
    }
    r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    r.Header.Add("Content-Length", strconv.Itoa(len(form.Encode())))

    res, err := client.Do(r)

    if err != nil {
        log.Printf("ERROR: %v\n", err)
	time.Sleep(2*time.Second)
	ch <- adif
	continue
    }

    log.Printf("Status: %v\n", res.Status)
    defer res.Body.Close()
    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
      log.Printf("ERROR: %v\n", err)
    }
    log.Println(string(body))
    default:
      time.Sleep(5*time.Second)
    }
  }
}

func main() {
 flag.Parse()
 if len(*key) < 1 {
    log.Fatal("API key must be provided (-k option)")
 }
 log.Printf("Reading from %s:%d\n", *ip, *port)
 ch := make(chan string)

 p := make([]byte, 2048)
 addr := net.UDPAddr{
      Port: *port,
      IP: net.ParseIP(*ip),
 }
 ser, err := net.ListenUDP("udp", &addr)
 if err != nil {
     fmt.Printf("ERROR: %v\n", err)
     return
 }
go send(ch)       
for {
   n, _, err := ser.ReadFromUDP(p)

   if err != nil {
         fmt.Printf("ERROR: %v", err)
         continue
   } 
 
   if n < 1 {
      continue
   }

   magic := binary.BigEndian.Uint32(p) 
   if magic != 0xadbccbda {
       fmt.Printf("ERROR: Unknown magic number: %x\n", magic)
       continue
   }
   var offset uint32 = 8
   payloadType := binary.BigEndian.Uint32(p[offset:]) 
   if payloadType == 12 {
       offset+=4
       idlen := binary.BigEndian.Uint32(p[offset:])
       offset+=4
       offset+=idlen
       adifLen := binary.BigEndian.Uint32(p[offset:])
       offset+=4
       adif := string(p[offset:offset+adifLen])
       ch <- adif
    }
 }
}
