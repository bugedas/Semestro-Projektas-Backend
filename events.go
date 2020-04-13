package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
)

type Event struct {
	gorm.Model
	Creator     User `gorm:"foreignkey:CreatorID"`
	CreatorID   uint
	Description string  `json: "description"`
	Location    string  `json: "location"`
	Users       []*User `gorm:"many2many:events_joined;"`
}

func CreateEvent(w http.ResponseWriter, r *http.Request) {
	var user User
	session, _ := sessionStore.Get(r, "auth-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	db.First(&user, session.Values["userID"].(uint))
	var newEvent Event
	newEvent.Creator = user
	err := json.NewDecoder(r.Body).Decode(&newEvent)

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	db.Model(&newEvent).AddForeignKey("creator_id", "users(id)", "RESTRICT", "RESTRICT")
	db.Create(&newEvent)
	w.WriteHeader(http.StatusCreated)
	return
}

func JoinEvent(w http.ResponseWriter, r *http.Request) {
	//Get user id from auth token
	session, _ := sessionStore.Get(r, "auth-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//Gets id from /events/{id}
	params := mux.Vars(r)
	eventID, err := strconv.Atoi(params["id"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//Get user and event seperately from provided IDs
	var user User
	db.First(&user, session.Values["userID"].(uint))

	var selectedEvent Event
	db.Preload("Users").First(&selectedEvent, "id = ?", eventID)

	//Check if event and user exist
	if selectedEvent.ID == 0 || user.ID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//Check if user is not the creator
	if user.ID == selectedEvent.CreatorID {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//Add user to event
	db.Model(&selectedEvent).Association("Users").Append(&user)

	w.WriteHeader(http.StatusOK)
	return
}

func LeaveEvent(w http.ResponseWriter, r *http.Request) {
	//Get user id from auth token
	session, _ := sessionStore.Get(r, "auth-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//decode event id from
	event := struct {
		ID int `json: "id"`
	}{}

	json.NewDecoder(r.Body).Decode(&event)

	var user User
	db.First(&user, session.Values["userID"].(uint))

	var selectedEvent Event
	db.Preload("Users").First(&selectedEvent, "id = ?", event.ID)

	if selectedEvent.ID == 0 || user.ID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	db.Model(&selectedEvent).Association("Users").Delete(&user)

	w.WriteHeader(http.StatusOK)
	return
}

func GetEvents(w http.ResponseWriter, r *http.Request) {
	keys := r.URL.Query()
	location := keys.Get("location")
	creatorID := keys.Get("creatorID")
	var events []Event

	tx := db.Table("events")

	if location != "" {
		tx = tx.Where("location = ?", location)
	}
	if creatorID != "" {
		tx = tx.Where("creator = ?", creatorID)
	}

	tx.Preload("Users").Find(&events)

	if len(events) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	JSONResponse(events, w)
	w.WriteHeader(http.StatusOK)
	return
}

func DeleteEvent(w http.ResponseWriter, r *http.Request) {
	//Loads creator id from authentication token
	session, _ := sessionStore.Get(r, "auth-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userID := session.Values["userID"].(uint)

	//Gets id from /events/{id}
	params := mux.Vars(r)
	eventID, err := strconv.Atoi(params["id"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//Loads event with joined users preloaded
	var event Event
	db.Preload("Users").Where("id = ?", eventID).First(&event)

	//checks if the user that is trying to delete event is its creator
	if event.CreatorID != userID {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//Deletes the record from database
	if db.Unscoped().Delete(&event).RecordNotFound() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	//Deletes associations (users that joined the event)
	db.Model(&event).Association("Users").Delete(&event.Users)

	w.WriteHeader(http.StatusOK)
	return
}
