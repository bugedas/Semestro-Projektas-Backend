package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/wader/gormstore"
	"golang.org/x/crypto/pbkdf2"
)

// Global variables -------------------------------------------
var db *gorm.DB
var sessionStore *gormstore.Store
var passwordRegex *regexp.Regexp

// ------------------------------------------------------------

type User struct {
	gorm.Model
	Username string
	Password string
	Salt     string
}

type Response struct {
	Data string `json:"data"`
}

//GenerateSalt creates a pseudorandom salt used in password salting
func GenerateSalt() string {
	salt, _ := uuid.NewV4()

	return hex.EncodeToString(salt.Bytes())
}

//GenerateSecurePassword generates a password using PBKDF2 standard
func GenerateSecurePassword(password string, salt string) string {
	hashedPassword := pbkdf2.Key([]byte(password), []byte(salt), 4096, 32, sha1.New)

	return hex.EncodeToString(hashedPassword)
}

//CheckNameAvailability checks if a username is available
func CheckNameAvailability(username string) error {
	var user User

	db.Find(&user, "username = ?", username)

	if user.Username != "" {
		return errors.New("Username exists")
	}

	return nil
}

//CreateNewAccount creates an account if the sent data
//is correctly formatted
func CreateNewAccount(username string, password string, repeatedPassword string) (httpStatus int, err error) {
	err = CheckNameAvailability(username)
	if err != nil {
		return http.StatusNotAcceptable, err
	}

	err = ComparePasswords(password, repeatedPassword)
	if err != nil {
		return http.StatusBadRequest, err
	}

	salt := GenerateSalt()
	hashedPassword := GenerateSecurePassword(password, salt)

	user := User{
		Username: username,
		Password: hashedPassword,
		Salt:     salt,
	}
	db.Debug().Create(&user)
	db.Save(&user)

	return http.StatusCreated, nil
}

//ComparePasswords checks that, while registering a new account,
//the password matches the repeated password, is atleast 8 characters long and
//contains at least one number and one capital letter
func ComparePasswords(passwordOne string, passwordTwo string) error {
	if passwordOne != passwordTwo {
		return errors.New("Passwords do not match")
	}

	if len(passwordOne) < 8 {
		return errors.New("Passwords too short")
	}

	if passwordRegex.MatchString(passwordOne) != true {
		return errors.New("Passwords needs to contain at least one number and one capital letter")
	}

	return nil
}

//RefreshToken refreshes authentication token WORK IN PROGRESS not even close to complete
func RefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshSession, _ := sessionStore.Get(r, "refresh-token")

	authSession, _ := sessionStore.Get(r, "auth-token")

	authSession.Values["userID"] = refreshSession.Values["userID"]
	authSession.Options.MaxAge = 60
	authSession.Save(r, w)

}

//JsonResponse sends a json response to user based on message
func JsonResponse(response interface{}, w http.ResponseWriter) {

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

	//Creates a table in the database for storing sessions
	//and sets a cleanup time
	sessionStore = gormstore.New(db, []byte("secret"))
	quit := make(chan struct{})
	go sessionStore.PeriodicCleanup(time.Minute, quit)

	HandleFunctions()

	log.Println("Server started on port 8000")
}
