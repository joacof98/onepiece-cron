// Package main provides the functionality of the CRON and Webscrapping jobs to get the latest OP chapter.
package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/gocolly/colly"
	"github.com/mileusna/crontab"
	"github.com/spf13/viper"
	gomail "gopkg.in/mail.v2"
)

const URL_ONEPIECE_CHAPTERS = "https://onepiecechapters.com/mangas/5/one-piece"

type OnePieceChapter struct {
	Name   string
	Number string
}

// sendEmailWithChapterData sends a email with the new chapter data to the emails of the users declared here.
func sendEmailWithChapterData(chapterName, chapterNumber string) {
	// get email password from .env
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error while reading config file %s", err)
	}

	pass, ok := viper.Get("EMAIL_PASSWORD").(string)
	if !ok {
		log.Fatalf("Invalid type assertion")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", "pandamancron@gmail.com")
	m.SetHeader("To", "joacofracchia@gmail.com")
	m.SetHeader("Subject", "A new chapter of One Piece arrived!")
	helloText := "Hi! Whats up! Our lord Oda delivered this to us:"
	endText := "You can read the chapter here: https://onepiecechapters.com/mangas/5/one-piece"
	m.SetBody("text/plain", helloText+"\n"+chapterNumber+" - "+chapterName+"\n\n"+endText)

	// Settings for SMTP server
	d := gomail.NewDialer("smtp.gmail.com", 587, "pandamancron@gmail.com", pass)

	// This is only needed when SSL/TLS certificate is not valid on server.
	// In production this should be set to false.
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err := d.DialAndSend(m); err != nil {
		log.Fatal(err)
		panic(err)
	} else {
		log.Println("Email sent!")
	}

	return
}

// crawlOnePieceChapter does the webscrapping in order to obtain the name and number of the last chapter ONLY if its new.
func crawlOnePieceChapter() {
	c := colly.NewCollector(
		colly.AllowedDomains("onepiecechapters.com"),
	)

	c.OnHTML(".col-span-2", func(element *colly.HTMLElement) {
		chapter := OnePieceChapter{}
		// search the latest chapter uploaded
		chapterList := element.DOM
		firstLink := chapterList.Find("a").First()
		chapter.Name = firstLink.Find(".text-gray-500").Text()
		chapter.Number = firstLink.Find(".text-lg.font-bold").Text()

		// compare to the latest number persisted
		content, err := ioutil.ReadFile("latest-chapter-number.txt")
		if err != nil {
			log.Fatal(err)
		}

		chapterWebNumber := strings.Split(chapter.Number, " ")[3]
		if string(content) == chapterWebNumber {
			log.Println("The chapter is not out yet :(")
		} else {
			// send a email notifying that the chapter is out!
			sendEmailWithChapterData(chapter.Name, chapter.Number)

			// update the last chapter crawled to the txt.
			data := []byte(chapterWebNumber)
			err := ioutil.WriteFile("latest-chapter-number.txt", data, 0777)
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	err := c.Visit(URL_ONEPIECE_CHAPTERS)
	if err != nil {
		log.Println(err)
	}
}

// main function, responsible for running the CRON.
func main() {
	viper.SetConfigFile(".env")
	cron := crontab.New()

	// run the cron job every thursday and once every hour from 11am to 23pm.
	err := cron.AddJob("0 11-23 * * */4", func() {
		crawlOnePieceChapter()
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Scanln()
}
