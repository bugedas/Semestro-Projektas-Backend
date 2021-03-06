package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/pbkdf2"
)

type User struct {
	ID          uint       `gorm:"primary_key"`
	CreatedAt   time.Time  `json:"-"`
	UpdatedAt   time.Time  `json:"-"`
	DeletedAt   *time.Time `json:"-"`
	Email       string     `gorm:"size:50;not null"`
	Username    string     `gorm:"size:30"`
	Gender      string     `gorm:"size:20"`
	Description string     `gorm:"size:255"`
	Password    string     `json:"-" gorm:"not null"`
	Salt        string     `json:"-" gorm:"size:64;not null"`
	Events      []*Event   `json:"-" gorm:"many2many:events_joined;"`
}

//RegisterPageHandler decodes user sent in data, verifies that
//it is formatted correctly, and tries to create an account in
//the database
func RegisterNewAccount(w http.ResponseWriter, r *http.Request) {
	//Creates a struct used to store data decoded from the body
	user := struct {
		Email          string `json: "email"`
		Username       string `json: "username"`
		Password       string `json: "password"`
		RepeatPassword string `json: "repeatPassword"`
		Gender         string `json: "gender"`
		Description    string `json: "description"`
	}{"", "", "", "", "", ""}

	err := json.NewDecoder(r.Body).Decode(&user)

	res, err := PerformUserDataChecks(user.Email, user.Password, user.RepeatPassword)

	w.WriteHeader(res)

	if err != nil {
		JSONResponse(struct{}{}, w)
		return
	}

	salt := GenerateSalt()
	hashedPassword := GenerateSecurePassword(user.Password, salt)

	newUser := User{
		Email:    user.Email,
		Username: user.Username,
		Password: hashedPassword,
		Gender:   user.Gender,
		Salt:     salt,
	}
	db.Debug().Create(&newUser)
	db.Save(&newUser)

	JSONResponse(struct{}{}, w)
	return
}

func Login(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "Access-token")

	if session.Values["userID"] != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}
	//Creates a struct used to store data decoded from the body
	userRequestData := struct {
		Email    string `json: "email"`
		Password string `json: "password"`
	}{"", ""}

	json.NewDecoder(r.Body).Decode(&userRequestData)

	var userDatabaseData User

	// Finds user by email in database, if no user, then returns "bad request"
	if db.Find(&userDatabaseData, "email = ?", userRequestData.Email).RecordNotFound() {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	hashedPassword := GenerateSecurePassword(userRequestData.Password, userDatabaseData.Salt)
	//checks if salted hashed password from database matches the sent in salted hashed password
	if hashedPassword != userDatabaseData.Password {
		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(struct{}{}, w)
		return
	}

	session = CreateAccessToken(userDatabaseData, session)
	session.Save(r, w)

	w.WriteHeader(http.StatusAccepted)
	JSONResponse(struct{}{}, w)
	return
}

func GetAccountInfo(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "Access-token")

	keys := r.URL.Query()
	id := keys.Get("id")

	var user User

	if id != "" {
		db.First(&user, id)
	} else if session.Values["userID"] != nil {
		db.First(&user, session.Values["userID"].(uint))
	} else {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	JSONResponse(user, w)
	w.WriteHeader(http.StatusOK)
	return
}

func EditPassword(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "Access-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	passwordData := struct {
		Password          string `json: "password"`
		NewPassword       string `json: "newPassword"`
		NewPasswordRepeat string `json: "newPasswordRepeat"`
	}{"", "", ""}

	json.NewDecoder(r.Body).Decode(&passwordData)

	var user User
	// Finds user by id in database, if no user, then returns "bad request"
	if db.Find(&user, "id = ?", session.Values["userID"]).RecordNotFound() {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	hashedPassword := GenerateSecurePassword(passwordData.Password, user.Salt)
	//checks if sent in password matches the database stored password
	if hashedPassword != user.Password {
		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(struct{}{}, w)
		return
	}

	//checks newPassword and newPasswordRepeat are the same
	err := ComparePasswords(passwordData.NewPassword, passwordData.NewPasswordRepeat)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	//Hashes new password and puts it in user
	newPassword := GenerateSecurePassword(passwordData.NewPassword, user.Salt)
	db.Model(&user).Updates(User{Password: newPassword})

	w.WriteHeader(http.StatusOK)
	JSONResponse(struct{}{}, w)
	return
}

func EditAccountInfo(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "Access-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	var user User
	tx := db.Where("id = ?", session.Values["userID"])

	var updatedUser User
	json.NewDecoder(r.Body).Decode(&updatedUser)

	if updatedUser.Username != "" {
		tx.Model(&user).Updates(User{Username: updatedUser.Username})
	}
	if updatedUser.Gender != "" {
		tx.Model(&user).Updates(User{Gender: updatedUser.Gender})
	}
	if updatedUser.Description != "" {
		tx.Model(&user).Updates(User{Description: updatedUser.Description})
	}
	tx.First(&user)

	w.WriteHeader(http.StatusOK)
	JSONResponse(struct{}{}, w)
	return
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
func CheckEmailAvailability(email string) error {
	var user User

	//if no record of the email is found, returns an error
	if !db.Find(&user, "email = ?", email).RecordNotFound() {
		return errors.New("Email exists")
	}

	return nil
}

//CreateNewAccount creates an account if the sent data
//is correctly formatted
func PerformUserDataChecks(email string, password string, repeatedPassword string) (httpStatus int, err error) {
	if emailRegex.MatchString(email) != true {
		return http.StatusNotAcceptable, errors.New("Bad email format")
	}

	err = CheckEmailAvailability(email)
	if err != nil {
		return http.StatusNotAcceptable, err
	}

	err = ComparePasswords(password, repeatedPassword)
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
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

func CreateAccessToken(user User, session *sessions.Session) *sessions.Session {
	//Access-token values
	session.Values["userID"] = user.ID
	session.Options.MaxAge = 60 * 60 * 24
	session.Options.HttpOnly = true
	return session
}

func IsLoggedIn(w http.ResponseWriter, r *http.Request) {
	session, err := sessionStore.Get(r, "Access-token")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	w.WriteHeader(http.StatusOK)
	JSONResponse(struct{}{}, w)
	return
}

func Logout(w http.ResponseWriter, r *http.Request) {
	sessionAccess, err := sessionStore.Get(r, "Access-token")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	sessionRefresh, err := sessionStore.Get(r, "Refresh-token")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	sessionAccess.Options.MaxAge = -1
	sessionRefresh.Options.MaxAge = -1

	sessionAccess.Save(r, w)
	sessionRefresh.Save(r, w)

	w.WriteHeader(http.StatusOK)
	JSONResponse(struct{}{}, w)
	return
}
