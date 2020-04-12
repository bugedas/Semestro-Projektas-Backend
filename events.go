package main

import (
	"encoding/json"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

type Event struct {
	gorm.Model
	Creator     int    `json: "creatorUserId"`
	Description string `json: "description"`
	Location    string `json: "location"`
}

func CreateEvent(w http.ResponseWriter, r *http.Request) {
	var newEvent Event

	err := json.NewDecoder(r.Body).Decode(&newEvent)

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	db.Create(&newEvent)
	w.WriteHeader(http.StatusCreated)
	return
}
func JoinEvent(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "auth-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	event := struct {
		ID int `json: "id"`
	}{}

	json.NewDecoder(r.Body).Decode(&event)

	var user User
	db.First(&user, session.Values["userID"].(uint))

	var eventD Event
	db.Preload("Users").First(&eventD, "id = ?", event.ID)

	if eventD.ID == 0 || user.ID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	db.Model(&eventD).Association("Users").Append(&user)

	w.WriteHeader(http.StatusOK)
	return
}
