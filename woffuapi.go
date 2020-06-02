package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type tokenJSON struct {
	Token string `json:"access_token"`
}

func getToken(user, pass string) (string, error) {
	client := &http.Client{}
	var data = strings.NewReader("grant_type=password&username=" + user + "&password=" + pass)
	req, err := http.NewRequest("POST", "https://app.woffu.com/token", data)
	if err != nil {
		return "", err
	}
	addCommonHeaders(req)
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("Origin", "https://app.woffu.com")
	req.Header.Set("Referer", "https://app.woffu.com/")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	t := &tokenJSON{}
	if err := json.Unmarshal(bodyText, t); err != nil {
		return "", err
	}
	return t.Token, nil
}

type userIDJSON struct {
	UserID int `json:"UserId"`
}

func getUserID(token string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://app.woffu.com/api/users", nil)
	if err != nil {
		return "", err
	}
	addCommonHeaders(req)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Referer", "https://app.woffu.com/")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	uid := &userIDJSON{}
	if err := json.Unmarshal(bodyText, uid); err != nil {
		return "", err
	}
	return strconv.Itoa(int(uid.UserID)), nil
}

func (w *woffu) login() error {
	// Get token
	token, err := getToken(w.User, w.Pass)
	if err != nil {
		return err
	}

	// Get user ID
	uid, err := getUserID(token)
	if err != nil {
		return err
	}
	w.WoffuToken = token
	w.WoffuUID = uid
	return nil
}

type eventJSON struct {
	ID   int    `json:"EventTypeId"`
	Name string `json:"Name"`
	Date string `json:"Date"`
}

func (w *woffu) getEvents() ([]eventJSON, error) {
	client := &http.Client{}
	dateTime := getDate()
	req, err := http.NewRequest("GET", "https://uniclau.woffu.com/api/users/321659/events?fromDate="+dateTime, nil)
	if err != nil {
		return nil, err
	}
	addCommonHeaders(req)
	addAuthHeaders(req, w.Corp, w.WoffuToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	events := []eventJSON{}
	if err := json.Unmarshal(bodyText, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (w *woffu) check() error {
	dateTime := getDate()
	client := &http.Client{}
	var data = strings.NewReader(`{"UserId":` + w.WoffuUID + `,"TimezoneOffset":-120,"StartDate":"` + dateTime + `","EndDate":"` + dateTime + `","DeviceId":"WebApp"}`)
	req, err := http.NewRequest("POST", "https://"+w.Corp+".woffu.com/api/svc/signs/signs", data)
	if err != nil {
		return err
	}
	addCommonHeaders(req)
	addAuthHeaders(req, w.Corp, w.WoffuToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Println(resp.StatusCode)
		fmt.Println(bodyText)
		return errors.New("Bad response")
	}
	return nil
}

func addCommonHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:76.0) Gecko/20100101 Firefox/76.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("DNT", "1")
	req.Header.Set("TE", "Trailers")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
}

func addAuthHeaders(req *http.Request, corp, token string) {
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Origin", "https://"+corp+".woffu.com")
	req.Header.Set("Referer", "https://"+corp+".woffu.com/")
	req.Header.Set("Cookie", "woffu.token="+token)
}

func getDate() string {
	t := time.Now()
	date := t.Format("2006-01-02")
	time := t.Format("15:04:05")
	_, zoneSecs := t.Zone()
	zoneHours := zoneSecs / 3600
	zoneStr := ""
	if zoneHours < 0 {
		zoneStr = "-"
		zoneHours *= -1
	} else {
		zoneStr = "+"
	}
	if zoneHours < 10 {
		zoneStr += "0"
	}
	hourStr := strconv.Itoa(zoneHours)
	zoneStr += hourStr + ":00"
	return date + "T" + time + zoneStr
}
