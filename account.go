package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/pbkdf2"
)

type User struct {
	gorm.Model
	Username string
	Password string
	Salt     string
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

//RegisterPageHandler decodes user sent in data, verifies that
//it is formatted correctly, and tries to create an account in
//the database
func RegisterNewAccount(w http.ResponseWriter, r *http.Request) {
	user := struct {
		Username       string `json: "username"`
		Password       string `json: "password"`
		RepeatPassword string `json: "repeatPassword"`
	}{"", "", ""}

	err := json.NewDecoder(r.Body).Decode(&user)

	res, err := PerformUserDataChecks(user.Username, user.Password, user.RepeatPassword)

	w.WriteHeader(res)

	if err != nil {
		response := Response{err.Error()}
		JsonResponse(response, w)
		return
	}

	salt := GenerateSalt()
	hashedPassword := GenerateSecurePassword(user.Password, salt)

	newUser := User{
		Username: user.Username,
		Password: hashedPassword,
		Salt:     salt,
	}
	db.Debug().Create(&newUser)
	db.Save(&newUser)

	response := Response{"Account created successfully"}
	JsonResponse(response, w)
	return
}

func Login(w http.ResponseWriter, r *http.Request) {
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
func PerformUserDataChecks(username string, password string, repeatedPassword string) (httpStatus int, err error) {
	err = CheckNameAvailability(username)
	if err != nil {
		return http.StatusNotAcceptable, err
	}

	err = ComparePasswords(password, repeatedPassword)
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusCreated, nil
}
