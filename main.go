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
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/wader/gormstore"
	"golang.org/x/crypto/pbkdf2"
)

// Global variables -------------------------------------------
var db *gorm.DB
var sessionStore *gormstore.Store
var passwordReg *regexp.Regexp

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

	if passwordReg.MatchString(passwordOne) != true {
		return errors.New("Passwords needs to contain at least one number and one capital letter")
	}

	return nil
}

//LandingPage comment
func LandingPage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello world"))
}

//RegisterPageHandler decodes user sent in data, verifies that
//it is formatted correctly, and tries to create an account in
//the database
func RegisterPageHandler(w http.ResponseWriter, r *http.Request) {
	user := struct {
		Username       string `json: "username"`
		Password       string `json: "password"`
		RepeatPassword string `json: "repeatPassword"`
	}{"", "", ""}

	err := json.NewDecoder(r.Body).Decode(&user)

	res, err := CreateNewAccount(user.Username, user.Password, user.RepeatPassword)

	w.WriteHeader(res)

	if err != nil {
		response := Response{err.Error()}
		JsonResponse(response, w)
		return
	}

	response := Response{"Account created successfully"}
	JsonResponse(response, w)
	return
}

//LoginPageHandler comment
func LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "auth-token")

	if session.Values["userID"] != nil {
		w.WriteHeader(http.StatusUnauthorized)
		response := Response{"Already logged in"}
		JsonResponse(response, w)
		return
	}

	userRequestData := struct {
		Username string `json: "username"`
		Password string `json: "password"`
	}{"", ""}

	json.NewDecoder(r.Body).Decode(&userRequestData)

	var userDatabaseData User

	db.Find(&userDatabaseData, "username = ?", userRequestData.Username)

	if userDatabaseData.Username == "" {
		w.WriteHeader(http.StatusUnauthorized)
		response := Response{"Bad credentials"}
		JsonResponse(response, w)
		return
	}

	hashedPassword := GenerateSecurePassword(userRequestData.Password, userDatabaseData.Salt)

	if hashedPassword != userDatabaseData.Password {
		w.WriteHeader(http.StatusUnauthorized)
		response := Response{"Bad credentials"}
		JsonResponse(response, w)
		return
	}

	session.Values["userID"] = userDatabaseData.ID

	session.Save(r, w)

	w.WriteHeader(http.StatusAccepted)
	response := Response{"Login successful"}
	JsonResponse(response, w)
	return
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

//ProtectedHandler is for testing if a user permissions
func ProtectedHandler(w http.ResponseWriter, r *http.Request) {

	session, _ := sessionStore.Get(r, "auth-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		response := Response{"You have to log in"}
		JsonResponse(response, w)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := Response{"Gained access to protected resource"}
	JsonResponse(response, w)
	return
}

func main() {
	//Regular expression for passwords to contain at least one capital letter and one number
	passwordReg = regexp.MustCompile(`([A-Z].*=?)([0-9].*=?)|([0-9].*=?)([A-Z].*=?)`)

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

	//Handles requested links
	r := mux.NewRouter()
	r.HandleFunc("/", LandingPage)
	r.HandleFunc("/login", LoginPageHandler).Methods("POST")
	r.HandleFunc("/register", RegisterPageHandler).Methods("POST")
	r.HandleFunc("/resource", ProtectedHandler)

	log.Println("Server started on port 8000")
	http.ListenAndServe(":8000", r)
}
