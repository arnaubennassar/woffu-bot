package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dipsycat/calendar-telegram-go"
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
)

func (w *woffu) runTelegramBot() error {
	bot, err := telegram.NewBotAPI(w.BotToken)
	if err != nil {
		return err
	}
	w.Bot = bot
	helpMsg := `Supported commands:
	/dontCheckIn: Manully add a day in which I won't check in / out.
	/skipList: Show the list of manually added days in which I won't check in / out.
	/checkInNow: Check in inmediatly. This won't affect the scheduled check in / out operations.
	/checkOutNow: Check out inmediatly. This won't affect the scheduled check in / out operations.
	/help: Show this message.`
	w.sendMessage("I'm online 😀. Here are my supported commands: \n" + helpMsg)
	now := time.Now()
	kbrdMonth := now.Month()
	kbrdYear := now.Year()
	go func() {
		u := telegram.NewUpdate(0)
		u.Timeout = 60
		updates, err := bot.GetUpdatesChan(u)
		if err != nil {
			w.sendError(err)
		}
		for update := range updates {
			fmt.Println("Message received: ", update)
			if update.CallbackQuery != nil && update.CallbackQuery.Message.Chat.ID == w.ChatID {
				// Handle keyboard response
				splitted := strings.Split(update.CallbackQuery.Data, ".")
				var keyboardCalendar telegram.InlineKeyboardMarkup
				if update.CallbackQuery.Data == ">" {
					// Next month
					keyboardCalendar, kbrdYear, kbrdMonth = calendar.HandlerNextButton(kbrdYear, kbrdMonth)
					msg := telegram.NewMessage(w.ChatID, "Choose day to add to skip list:")
					msg.ReplyMarkup = keyboardCalendar
					bot.Send(msg)
				} else if update.CallbackQuery.Data == "<" {
					// Previous month
					keyboardCalendar, kbrdYear, kbrdMonth = calendar.HandlerPrevButton(kbrdYear, kbrdMonth)
					msg := telegram.NewMessage(w.ChatID, "Choose day to add to skip list:")
					msg.ReplyMarkup = keyboardCalendar
					bot.Send(msg)
				} else if len(splitted) == 3 {
					// Add date to skip list
					year, err := strconv.Atoi(splitted[0])
					if err != nil {
						w.sendError(errors.New("Unexpected message, not adding date"))
						continue
					}
					month, err := strconv.Atoi(splitted[1])
					if err != nil {
						w.sendError(errors.New("Unexpected message, not adding date"))
						continue
					}
					day, err := strconv.Atoi(splitted[2])
					if err != nil {
						w.sendError(errors.New("Unexpected message, not adding date"))
						continue
					}
					if year < now.Year() ||
						year == now.Year() && month < int(now.Month()) ||
						year == now.Year() && month == int(now.Month()) && day < now.Day() {
						w.sendMessage("⚠ This date has already passed, choose a date in the future ⚠")
						continue
					}
					// Check if not already added
					found := false
					for _, existingDate := range w.SkipList {
						if existingDate == update.CallbackQuery.Data {
							w.sendMessage("⚠ " + update.CallbackQuery.Data + " is already added in the /skipList ⚠")
							found = true
							break
						}
					}
					if !found {
						// Add to skip list
						w.SkipList = append(w.SkipList, update.CallbackQuery.Data)
						// Message skip item added
						w.sendMessage("Added 📆 " + update.CallbackQuery.Data + " 📆 to the skip list. You can edit the list with the command /skipList")
					}
				} else if len(splitted) == 4 && splitted[0] == "delete" {
					deleteDate := splitted[1] + "." + splitted[2] + "." + splitted[3]
					found := false
					for i, storedDate := range w.SkipList {
						if storedDate == deleteDate {
							w.SkipList[i] = w.SkipList[len(w.SkipList)-1]
							w.SkipList = w.SkipList[:len(w.SkipList)-1]
							w.sendMessage("❌ " + deleteDate + " deleted from list ❌")
							found = true
							break
						}
					}
					if !found {
						w.sendMessage("⚠ " + deleteDate + " not found. Check the updated /skipList ⚠")
					}
					continue
				}
			}
			if update.Message == nil { // ignore any non-Message Updates
				continue
			}
			if update.Message.Chat.ID != w.ChatID { // message from unexpected chat
				w.sendMessage(
					"I've received a message from an unexpected chat.\nChat id: `" +
						strconv.Itoa(int(update.Message.Chat.ID)) +
						"`\nMessage: `" + update.Message.Text + "`",
				)
				continue
			}

			// TODO: handle {add holyday, change check in/out}
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "help":
					w.sendMessage(helpMsg)
				case "dontCheckIn":
					now = time.Now()
					kbrdMonth = now.Month()
					kbrdYear = now.Year()
					keyboardCalendar := calendar.GenerateCalendar(kbrdYear, kbrdMonth)
					msg := telegram.NewMessage(w.ChatID, "Choose day to add to skip list:")
					msg.ReplyMarkup = keyboardCalendar
					bot.Send(msg)
				case "checkInNow":
					if err := w.check(); err != nil {
						w.sendError(err)
					} else {
						w.sendMessage("Checked in successfuly")
					}
				case "checkOutNow":
					if err := w.check(); err != nil {
						w.sendError(err)
					} else {
						w.sendMessage("Checked out successfuly")
					}
				case "skipList":
					// Clean list (past days)
					today := getCurrentDate()
					tmpList := []string{}
					for _, date := range w.SkipList {
						fmt.Println(date)
						if date >= today {
							fmt.Println("yas")
							tmpList = append(tmpList, date)
						}
					}
					w.SkipList = tmpList
					if len(w.SkipList) == 0 {
						w.sendMessage("There are no dates in the skip list. You can add days in which I won't check in / out with the command /dontCheckIn")
					} else {
						keyboard := telegram.InlineKeyboardMarkup{}
						// Create keyboard that allows day deletion
						for _, date := range w.SkipList {
							row := []telegram.InlineKeyboardButton{}
							row = append(row, telegram.NewInlineKeyboardButtonData(date, "foo"))
							row = append(row, telegram.NewInlineKeyboardButtonData("❌", "delete."+date))
							keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
						}
						msg := telegram.NewMessage(w.ChatID, "Days in which I won't check in / out:")
						msg.ReplyMarkup = keyboard
						if _, err := bot.Send(msg); err != nil {
							w.sendError(err)
						}
					}
				default:
					w.sendMessage("I don't know that command. Do you need /help?")
				}
			}

		}
		w.sendMessage("I'm offline 🙃")
	}()
	return nil
}

func (w *woffu) sendMessage(msg string) error {
	if w.Bot == nil {
		fmt.Println(msg)
		return nil
	}
	fmt.Println("Sending message:", msg)
	_, err := w.Bot.Send(telegram.NewMessage(w.ChatID, msg))
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (w *woffu) sendError(err error) error {
	return w.sendMessage("Something went wrong 😰😱:\n\n🔥💥💣 " + err.Error() + " 💣💥🔥")
}

func getCurrentDate() string {
	currentTime := time.Now()
	year := strconv.Itoa(currentTime.Year())
	month := strconv.Itoa(int(currentTime.Month()))
	day := strconv.Itoa(currentTime.Day())
	if len(month) == 1 {
		month = "0" + month
	}
	if len(day) == 1 {
		day = "0" + day
	}
	return year + "." + month + "." + day
}
