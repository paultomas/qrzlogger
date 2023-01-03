package qrz
import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const LOGBOOK_URL = "https://logbook.qrz.com/api"
var ErrAlreadyExists = errors.New("duplicate entry")

type Client struct {
	key string
}

func NewClient(key string) *Client {
	return &Client{key: key}
}

func (c *Client) Upload(adif string) error {
	form := url.Values{}
	form.Set("ACTION", "INSERT")
	form.Set("KEY", c.key)
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
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "LOGID") {
	   if strings.Contains(bodyStr, "&REASON=Unable to add QSO to database: duplicate") {
               return ErrAlreadyExists
	   }
	   return errors.New(string(body))
	}
	log.Printf("Logged:\n%s\n", adif)
	log.Println(string(body))
	return nil
}
