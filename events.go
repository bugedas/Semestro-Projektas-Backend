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
