package main

import (
	"errors"
	"log"	
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)
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
