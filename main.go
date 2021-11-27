package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var configuration Configuration

type wallet map[string]float64
type myDb map[int64]wallet
type binanceResp struct {
	Price float64 `json:"price,string"`
	Code  int64   `json:"code"`
}

var db = myDb{}

func init() {
	configuration = GetConfig()
}

func main() {
	bot, err := tgbotapi.NewBotAPI(configuration.ApiKey)
	if err != nil {
		log.Panic(err)
	}

	dbBytes, err := ioutil.ReadFile(configuration.DbFileName)
	if err != nil {
		log.Fatalf("can't read db file: %s", err)
	}
	err = json.Unmarshal(dbBytes, &db)
	if err != nil {
		log.Fatalf("can't unmarchaled db file: %s", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		chatId := update.Message.Chat.ID

		command := strings.Split(update.Message.Text, " ")
		typeCommand := command[0]

		switch typeCommand {
		case "ADD":
			if len(command) == 3 {
				if len(command[1]) > 0 {
					currency := command[1]
					money, err := strconv.ParseFloat(command[2], 64)
					if err != nil {
						Send(bot, chatId, "Ошибка получения значения")
						log.Printf("error parse float: %s", err)
						continue
					}
					if _, ok := db[chatId]; !ok {
						db[chatId] = wallet{}
					}

					db[chatId][currency] += money

					Send(bot, chatId, fmt.Sprintf("Баланс: %s %f", currency, db[chatId][currency]))
					Save(db)
				} else {
					Send(bot, chatId, "Валюта не может быть пустой")
					log.Printf("error len currency zero")
				}
			} else {
				Send(bot, chatId, "Команда ADD имеет формат ADD <currency> <money>")
				log.Printf("error len command arguments")
			}
		case "SUB":
			if len(command) == 3 {
				if len(command[1]) > 0 {
					currency := command[1]
					money, err := strconv.ParseFloat(command[2], 64)
					if err != nil {
						Send(bot, chatId, "Ошибка получения значения")
						log.Printf("error parse float: %s", err)
						continue
					}
					if money > 0 {
						if _, ok := db[chatId]; !ok {
							db[chatId] = wallet{}
						}

						if db[chatId][currency] > money {
							db[chatId][currency] -= money

							Send(bot, chatId, fmt.Sprintf("Баланс: %s %f", currency, db[chatId][currency]))
							Save(db)
						} else {
							Send(bot, chatId, fmt.Sprintf("Баланс меньше удаляемого количество, пожалуйста измените параметр"))
						}
					} else {
						Send(bot, chatId, "Валюта не может быть минусовой")
					}
				} else {
					Send(bot, chatId, "Валюта не может быть пустой")
					log.Printf("error len currency zero")
				}
			} else {
				Send(bot, chatId, "Команда SUB имеет формат SUB <currency> <money>")
				log.Printf("error len command arguments")
			}
		case "DEL":
			if len(command) == 2 {
				currency := command[1]
				delete(db[chatId], currency)
				Send(bot, chatId, "Валюта удалена")
				Save(db)
			} else {
				Send(bot, chatId, "Команда DEL имеет формат DEL <currency>")
				log.Printf("error len command arguments")
			}
		case "SHOW":
			msg := "Баланс:\n"
			var usdSumm float64
			rub, err := getRub()
			if err != nil {
				Send(bot, chatId, "Не смог узнать цену валюты")
				log.Printf("error get rub price: %s", err)
			}
			for currency, money := range db[chatId] {
				coinPrice, err := getPrice(currency)
				if err != nil {
					Send(bot, chatId, "Не смог узнать цену валюты")
					log.Printf("error get price: %s", err)
				}
				usdSumm += money * coinPrice
				msg += fmt.Sprintf("%s: %.2f [%.2f USD] [%.2f ₽]\n", currency, money, money*coinPrice, money*coinPrice*rub)
			}
			msg += fmt.Sprintf("Сумма в USD: %.2f\n", usdSumm)
			msg += fmt.Sprintf("Сумма в ₽: %.2f", usdSumm*rub)
			Send(bot, chatId, msg)

		default:
			Send(bot, chatId, "Не известная команда: доступны команды ADD, SUB, DEL, SHOW")
		}
	}
}

// Send метод для отправки сообщений в телеграм
func Send(bot *tgbotapi.BotAPI, chatId int64, msg string) bool {
	sendingMsg := tgbotapi.NewMessage(chatId, msg)
	_, err := bot.Send(sendingMsg)
	if err != nil {
		log.Printf("can't send to telegram bot message: %s\n", err)
		return false
	}

	return true
}

func Save(data myDb) {
	var jsonData interface{}
	jsonData = data
	writerData, err := json.Marshal(jsonData)
	if err != nil {
		log.Fatalf("can't marshaled: %s", err)
	}
	err = ioutil.WriteFile("db.json", writerData, 0644)
	if err != nil {
		log.Fatalf("can't write data to db file: %s", err)
	}
}

func getPrice(coin string) (price float64, err error) {
	resp, err := http.Get(fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%sUSDT", coin))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var jsonResp binanceResp
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		return
	}

	if jsonResp.Code != 0 {
		err = errors.New("invalid currency")
		return
	}

	return jsonResp.Price, nil
}

func getRub() (price float64, err error) {
	resp, err := http.Get("https://api.binance.com/api/v3/ticker/price?symbol=USDTRUB")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var jsonResp binanceResp
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		return
	}

	if jsonResp.Code != 0 {
		err = errors.New("invalid currency")
		return
	}

	return jsonResp.Price, nil
}
