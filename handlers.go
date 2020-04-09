package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

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

//LandingPageHandler comment
func LandingPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello world"))
}

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

func HandleFunctions() {
	r := mux.NewRouter()
	r.HandleFunc("/", LandingPageHandler)
	r.HandleFunc("/login", LoginPageHandler).Methods("POST")
	r.HandleFunc("/register", RegisterPageHandler).Methods("POST")
	r.HandleFunc("/resource", ProtectedHandler)
	http.ListenAndServe(":8000", r)
}
