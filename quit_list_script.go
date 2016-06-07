package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"github.com/jinzhu/gorm"
	"encoding/csv"
	"io"
	//"github.com/NewtopiaCI/common/log"
	"github.com/NewtopiaCI/common/models"
	"github.com/NewtopiaCI/common/database"
)

type QuitListTable struct {
	FirstName   string    `json:"first_name"`  	// col 0
	LastName  	string    `json:"last_name"`  	// col 1
	Email  		string    `json:"email"`  		// col 2
	Company		string    `json:"company"`  	// col 3
	Date		string    `json:"date"`  		// col 4
	Description string    `json:"description"`  // col 5
	Reason		string    `json:"reason"`  		// col 6
	State		string    `json:"state"`  		// col 7
	FullName	string    `json:"full_name"`  	// col 8
}

func init() {
	//Set up logging connection for common/log
	// configLog := log.LogConfiguration{
	// 	Tag:       "producer_spire_script",
	// 	Network:   "tcp",
	// 	DBint:     1,
	// 	LogServer: "192.168.99.100:6379",
	// 	IsDebug:   false,
	// 	Version:   "1",
	// }
	// log.SetConfiguration(configLog)

	// Set up DB connection for common/database, as models.User functions use that configuration
	dbConfig := database.DBConfiguration{
		Host: 		"localhost",
		Port: 		5432,
		SSLMode: 	"disable",
		User:     	"devadm",
		Password:   "cHangeIT",
		Database:   "devlocal_app",
	}
	database.SetAppDatabase(dbConfig)
}

func main(){
	log.Print("Start Script")
	extractFile("quit_list.csv")
}

func extractFile(filename string){
	log.Print("Newtopia Quit List - Start Extraction")
	lineCount := 0

	//Check if CSV file
	ext := filepath.Ext(filename)
	if ext != ".csv" {
		err := errors.New("Error: Input file is not .csv")
		log.Print("Newtopia Quit List script error: ", err)
		return
	}

	file, err := os.Open(filename)
	if err != nil {
		log.Print("Newtopia Quit List script error: ", err)
		return
	}
	defer file.Close()

	r := csv.NewReader(file)

	for i := 0; ; i++ {
		var data QuitListTable
		lineCount = i

		row, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Print("Newtopia Quit List script error: ", err)
			return
		}

		// Skip first row (headings)
		if i == 0 {
			continue
		}

		data = QuitListTable {
			FirstName:     	row[0],
			LastName:		row[1],
			Email:  		row[2],
			Company:  		row[3],
			Date:			row[4],
			Description:  	row[5],
			Reason:     	row[6],
			State:  		row[7],
			FullName:		row[8],
		}

		//Grab User ID via email
		var userEmail models.UserEmail
		err = database.App.Where("email = ?", data.Email).Find(&userEmail).Error
		if err != nil {
			log.Print("Newtopia Quit List script error: ", err, " ~ Name: ", data.FirstName, " ", data.LastName, " , Email: ", data.Email)
			continue
		}

		// First soft-delete previous state of same type
		err = database.App.Where("user_id = ? AND type = ?", userEmail.UserId, "status").Delete(models.UserState{}).Error
		if err != nil && err != gorm.ErrRecordNotFound{
			log.Print("db error clearing previous user state: ", err, " ~ Name: ", data.FirstName, " ", data.LastName, " , Email: ", data.Email)
			continue
		}

		// Set new state
		var userState models.UserState
		userState = models.UserState{
			UserId: models.UUID{userEmail.UserId.UUID},
			Type:   "status",
			State:  data.State,
			Reason: data.Reason,
			Description: data.Date + " " + data.Description,
		}

		err = database.App.Create(&userState).Error
		if err != nil {
			log.Print("db error creating new user state", err, " ~ Name: ", data.FirstName, " ", data.LastName, " , Email: ", data.Email)
			continue
		}

	}

	log.Print("Newtopia Quit List script: Finished ", lineCount, " lines.")
	return
}
