package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

type Event struct {
	ID          uint       `json: "-" gorm:"primary_key"`
	CreatedAt   time.Time  `json: "-"`
	UpdatedAt   time.Time  `json: "-"`
	DeletedAt   *time.Time `json: "-"`
	Creator     User       `gorm:"foreignkey:CreatorID"`
	CreatorName string
	CreatorID   uint
	Description string    `json: "description"`
	Sport       string    `json: "sport"`
	Location    string    `json: "location"`
	StartTime   time.Time `json: "startTime"`
	EndTime     time.Time `json: "endTime"`
	Limit       int       `json: "limit"`
	Users       []*User   `gorm:"many2many:events_joined;"`
}

func CreateEvent(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "Access-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(struct{}{}, w)
		return
	}
	// Get the user that is creating the event
	var user User
	db.First(&user, session.Values["userID"].(uint))

	var newEvent Event
	// Get event data from json body
	err := json.NewDecoder(r.Body).Decode(&newEvent)
	newEvent.Creator = user
	newEvent.CreatorName = user.Username
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Create an association between creator_id and a users id
	db.Model(&newEvent).AddForeignKey("creator_id", "users(id)", "RESTRICT", "RESTRICT")
	// Create event
	if db.Create(&newEvent).Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSONResponse(struct{}{}, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	JSONResponse(struct{}{}, w)
	return
}

func JoinEvent(w http.ResponseWriter, r *http.Request) {
	//Get user id from auth token
	session, _ := sessionStore.Get(r, "Access-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(struct{}{}, w)
		return
	}

	//Gets id from /events/{id}/users
	params := mux.Vars(r)
	eventID, err := strconv.Atoi(params["id"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	//Get user and event from provided IDs
	var user User
	db.First(&user, session.Values["userID"].(uint))

	var selectedEvent Event
	db.Preload("Users").First(&selectedEvent, "id = ?", eventID)

	if selectedEvent.Limit != 0 {
		if selectedEvent.Limit <= len(selectedEvent.Users) {
			w.WriteHeader(http.StatusInsufficientStorage)
			JSONResponse(struct{}{}, w)
			return
		}
	}
	//Check if event and user exist
	if selectedEvent.ID == 0 || user.ID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	//Check if user is not the creator
	if user.ID == selectedEvent.CreatorID {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	//Add user to event
	db.Model(&selectedEvent).Association("Users").Append(&user)

	w.WriteHeader(http.StatusOK)
	JSONResponse(struct{}{}, w)
	return
}

func LeaveEvent(w http.ResponseWriter, r *http.Request) {
	//Get user id from auth token
	session, _ := sessionStore.Get(r, "Access-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(struct{}{}, w)
		return
	}

	//Gets id from /events/{id}/users
	params := mux.Vars(r)
	eventID, err := strconv.Atoi(params["id"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	//Get user and event from provided IDs
	var user User
	db.First(&user, session.Values["userID"].(uint))

	var selectedEvent Event
	db.Preload("Users").First(&selectedEvent, "id = ?", eventID)

	//Check if event and user exist
	if selectedEvent.ID == 0 || user.ID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	//Check if user is not the creator
	if user.ID == selectedEvent.CreatorID {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	// Delete user from an event
	db.Model(&selectedEvent).Association("Users").Delete(&user)

	w.WriteHeader(http.StatusOK)
	JSONResponse(struct{}{}, w)
	return
}

func GetEvents(w http.ResponseWriter, r *http.Request) {
	// Gets filtering keys from url. e.x ?location=kaunas&creatorId=1
	keys := r.URL.Query()
	location := keys.Get("location")
	creatorID := keys.Get("creatorID")
	var events []Event

	// Preloads user and creator tables for use in event response
	tx := db.Preload("Users").Preload("Creator")

	// If a certain tag is not null, it is used to filter events
	if location != "" {
		tx = tx.Where("location = ?", location)
	}
	if creatorID != "" {
		tx = tx.Where("creator_id = ?", creatorID)
	}
	// Finds events based on given parameters
	tx.Find(&events)

	// If no events exist, return Bad request
	if len(events) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	w.WriteHeader(http.StatusOK)
	JSONResponse(events, w)
	return
}

func DeleteEvent(w http.ResponseWriter, r *http.Request) {
	//Loads creator id from authentication token
	session, _ := sessionStore.Get(r, "Access-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(struct{}{}, w)
		return
	}
	userID := session.Values["userID"].(uint)

	//Gets id from /events/{id}
	params := mux.Vars(r)
	eventID, err := strconv.Atoi(params["id"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	//Loads event with joined users preloaded
	var event Event
	db.Preload("Users").Where("id = ?", eventID).First(&event)

	//checks if the user that is trying to delete event is its creator
	if event.CreatorID != userID {
		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(struct{}{}, w)
		return
	}

	//Deletes the record from database
	if db.Unscoped().Delete(&event).RecordNotFound() {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	//Deletes associations (users that joined the event)
	db.Model(&event).Association("Users").Delete(&event.Users)

	w.WriteHeader(http.StatusOK)
	JSONResponse(struct{}{}, w)
	return
}

func EditEvent(w http.ResponseWriter, r *http.Request) {
	//Loads creator id from authentication token
	session, _ := sessionStore.Get(r, "Access-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(struct{}{}, w)
		return
	}
	userID := session.Values["userID"].(uint)

	//Gets id from /events/{id}
	params := mux.Vars(r)
	eventID, err := strconv.Atoi(params["id"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	//Loads event with joined users preloaded
	var event Event
	tx := db.Preload("Users").Where("id = ?", eventID).First(&event)

	//checks if the user that is trying to delete event is its creator
	if event.CreatorID != userID {
		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(struct{}{}, w)
		return
	}

	var updatedEvent Event
	json.NewDecoder(r.Body).Decode(&updatedEvent)

	if updatedEvent.Description != "" {
		tx.Model(&event).Updates(Event{Description: updatedEvent.Description})
	}
	if updatedEvent.StartTime.Year() != 1 {
		tx.Model(&event).Updates(Event{StartTime: updatedEvent.StartTime})
	}
	if updatedEvent.EndTime.Year() != 1 {
		tx.Model(&event).Updates(Event{EndTime: updatedEvent.EndTime})
	}
	if updatedEvent.Limit != 0 {
		tx.Model(&event).Updates(Event{Limit: updatedEvent.Limit})
	}
	// //Edits the record in database
	// if tx.Model(&event).Updates(Event{Description: updatedEvent.Description}).RowsAffected == 0 {
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	JSONResponse(struct{}{}, w)
	// 	return
	// }

	w.WriteHeader(http.StatusOK)
	JSONResponse(struct{}{}, w)
	return
}
