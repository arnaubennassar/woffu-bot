package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
)

type woffu struct {
	User                 string
	Pass                 string
	Corp                 string
	BotToken             string
	ChatID               int64
	CheckInHour          int
	CheckInMinute        int
	CheckOutHour         int
	CheckOutMinute       int
	SeconsOfInprecission int
	WorkingEventIDs      []int
	Bot                  *telegram.BotAPI
	WoffuToken           string
	WoffuUID             string
	SkipList             []string
}

func main() {
	// Load config
	w, err := newBot()
	if err != nil {
		panic(err)
	}

	// Get credentials
	err = w.login()
	if err != nil {
		w.sendError(errors.New("Error login: " + err.Error() + ". This will cause panic!"))
		panic(err)
	}

	// Endless loop, because work does never end
	errCount := 0
	for {
		// Do nothing until check in/out time
		isCheckIn, inprecission := w.sleepTillNext()
		// Get events
		evs, err := w.getEvents()
		if err != nil {
			// Maybe token has expired, renew credentials and retry
			if errCount == 1 {
				w.sendError(errors.New(err.Error() + ". This will cause panic!"))
				panic("Too many consecutive errors")
			} else {
				w.sendError(err)
			}
			err = w.login()
			if err != nil {
				w.sendError(err)
			}
			errCount++
			time.Sleep(10 * time.Second)
			continue
		}

		// Check in/out if it's a working day
		isWorkingDay := false
		for _, id := range w.WorkingEventIDs {
			if evs[0].ID == id {
				isWorkingDay = true
				break
			}
		}
		isSkipDay := false
		today := getCurrentDate()
		for i, skipDate := range w.SkipList {
			if skipDate == today {
				isSkipDay = true
				w.SkipList[i] = w.SkipList[len(w.SkipList)-1]
				w.SkipList = w.SkipList[:len(w.SkipList)-1]
			}
		}
		if isWorkingDay && !isSkipDay {
			// It's a working day
			if err := w.check(); err != nil {
				w.sendError(err)
			} else {
				errCount = 0
				if isCheckIn {
					w.sendMessage("Checked in successfully")
				} else {
					w.sendMessage("Checked out successfully")
				}
			}
		} else {
			// It's NOT a working day
			if isCheckIn {
				if isSkipDay {
					w.sendMessage("You told me to not check in today, so I won't")
				} else {
					w.sendMessage("Enjoy your free day 😎")
				}
			}
			errCount = 0
			fmt.Println("Free day, not checking in/out")
		}
		if inprecission < 0 {
			time.Sleep(inprecission)
		}
		time.Sleep(time.Minute)
	}
}

func newBot() (*woffu, error) {
	// Load config
	w, err := loadConfig()
	if err != nil {
		return nil, err
	}

	// Run bot
	err = w.runTelegramBot()
	return w, err
}

func loadConfig() (*woffu, error) {
	user := os.Getenv("WOFFU_USER")
	if user == "" {
		return nil, errors.New("WOFFU_USER env value is mandatory")
	}
	pass := os.Getenv("WOFFU_PASS")
	if pass == "" {
		return nil, errors.New("WOFFU_PASS env value is mandatory")
	}
	corp := os.Getenv("CORP")
	if corp == "" {
		return nil, errors.New("CORP env value is mandatory")
	}
	botToken := os.Getenv("BOT")
	chatID, err := strconv.Atoi(os.Getenv("CHAT"))
	if botToken != "" && err != nil {
		return nil, err
	}
	parseTime := func(s string) (int, int, error) {
		splitted := strings.Split(s, ":")
		hour, err := strconv.Atoi(splitted[0])
		if err != nil {
			return 0, 0, err
		}
		minute, err := strconv.Atoi(splitted[1])
		if err != nil {
			return 0, 0, err
		}
		if hour < 0 || hour > 59 || minute < 0 || minute > 59 {
			return 0, 0, errors.New("Wrong value")
		}
		return hour, minute, nil
	}
	checkInHour, checkInMinute, err := parseTime(os.Getenv("CHECKIN"))
	if err != nil {
		return nil, errors.New("Error parsing CHECKIN: " + err.Error())
	}
	checkOutHour, checkOutMinute, err := parseTime(os.Getenv("CHECKOUT"))
	if err != nil {
		return nil, errors.New("Error parsing CHECKOUT: " + err.Error())
	}
	splitted := strings.Split(os.Getenv("WORKINGDAYIDS"), ",")
	workingIDs := []int{}
	for _, idStr := range splitted {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return nil, err
		}
		workingIDs = append(workingIDs, id)
	}
	secs, err := strconv.Atoi(os.Getenv("IMPRECISSION"))
	if err != nil {
		secs = 0
	}
	return &woffu{
		User:                 user,
		Pass:                 pass,
		Corp:                 corp,
		BotToken:             botToken,
		ChatID:               int64(chatID),
		CheckInHour:          checkInHour,
		CheckInMinute:        checkInMinute,
		CheckOutHour:         checkOutHour,
		CheckOutMinute:       checkOutMinute,
		SeconsOfInprecission: secs,
		WorkingEventIDs:      workingIDs,
		SkipList:             []string{},
	}, nil
}

// sleepTillNext returns true if it has sleep until check in time, false for check out
func (w *woffu) sleepTillNext() (bool, time.Duration) {
	currentTime := time.Now()
	fmt.Println("current time: ", currentTime.Hour(), ":", currentTime.Minute())
	sleepHours := 0
	sleepMinutes := 0
	isCheckIn := true
	if currentTime.Minute() <= w.CheckInMinute {
		sleepMinutes = w.CheckInMinute - currentTime.Minute()
	} else {
		sleepMinutes = (currentTime.Minute() - w.CheckInMinute) * -1
	}
	if currentTime.Hour() < w.CheckInHour || (currentTime.Hour() == w.CheckInHour && currentTime.Minute() <= w.CheckInMinute) {
		fmt.Println("not started day case")
		sleepHours = w.CheckInHour - currentTime.Hour()
	} else if currentTime.Hour() > w.CheckOutHour || (currentTime.Hour() == w.CheckOutHour && currentTime.Minute() > w.CheckOutMinute) {
		fmt.Println("finished day case")
		sleepHours = 24 - currentTime.Hour() + w.CheckInHour
	} else {
		fmt.Println("in the office case")
		isCheckIn = false
		sleepHours = w.CheckOutHour - currentTime.Hour()
		if currentTime.Minute() <= w.CheckOutMinute {
			sleepMinutes = w.CheckOutMinute - currentTime.Minute()
		} else {
			sleepMinutes = (currentTime.Minute() - w.CheckOutMinute) * -1
		}
	}
	inprecission := time.Duration(rand.Intn(w.SeconsOfInprecission)-w.SeconsOfInprecission/2) * time.Second
	sleepTime := time.Duration(sleepHours)*time.Hour + time.Minute*time.Duration(sleepMinutes) + inprecission
	fmt.Println("Sleeping for: ", sleepTime)
	time.Sleep(sleepTime)
	return isCheckIn, inprecission
}
