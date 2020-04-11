package main

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/wader/gormstore"
)

// Global variables -------------------------------------------
var db *gorm.DB
var sessionStore *gormstore.Store
var passwordRegex *regexp.Regexp

// ------------------------------------------------------------

type Response struct {
	Data string `json:"data"`
}

//JSONResponse sends a json response to user based on message
func JSONResponse(response interface{}, w http.ResponseWriter) {

	json, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func main() {
	//Regular expression for passwords to contain at least one capital letter and one number
	passwordRegex = regexp.MustCompile(`([A-Z].*=?)([0-9].*=?)|([0-9].*=?)([A-Z].*=?)`)

	log.Println("Opening database")
	var err error
	db, err = gorm.Open("mysql", "root:pass@tcp(localhost:3306)/semestroprojektasktu2020?parseTime=true&parseTime=True")

	if err != nil {
		log.Println(err.Error())
	}

	//Checks if Users table exists, if it does not, creates one
	if !db.HasTable(&User{}) {
		db.CreateTable(&User{})
	}
	if !db.HasTable(&Event{}) {
		db.CreateTable(&Event{})
	}

	//Creates a table in the database for storing sessions
	//and sets a cleanup time
	sessionStore = gormstore.New(db, []byte("secret"))
	quit := make(chan struct{})
	go sessionStore.PeriodicCleanup(time.Minute, quit)

	HandleFunctions()
}
