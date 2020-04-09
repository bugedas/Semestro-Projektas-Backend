package main

import (
	_ "github.com/go-sql-driver/mysql"
)

type Event struct {
	Description string `json: "description"`
	City        string `json: "repeatPassword"`
	Location    string `json "location"`
}

func CreateEvent() {
	var newEvent Event

	db.Create(&newEvent)
}

func DeleteEvent() {
	var event Event
	var id int

	db.Where("id = ?", id).Delete(&event)
}
