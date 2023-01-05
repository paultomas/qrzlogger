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
	keyValLines := strings.Split(bodyStr, "&")
	keyVals := make(map[string]string)
	for _, v := range keyValLines {
	   kv := strings.Split(v, "=")
	   if len(kv) > 1 {
              keyVals[kv[0]] = kv[1]
	   }
	}
	if keyVals["LOGID"] == "" {
	     if strings.Contains(keyVals["REASON"], "duplicate") {
                 return ErrAlreadyExists		    
	     }
	     log.Printf("ERROR: %s\n", bodyStr)
	     return errors.New(keyVals["REASON"])
	}
	log.Printf("Logged:\n%s\n", adif)
	return nil
}
