package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

//ProtectedHandler is for testing if a user permissions
func ProtectedHandler(w http.ResponseWriter, r *http.Request) {

	session, _ := sessionStore.Get(r, "auth-token")

	if session.Values["userID"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		response := Response{"You have to log in"}
		JSONResponse(response, w)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := Response{"Gained access to protected resource"}
	JSONResponse(response, w)
	return
}

//LandingPage comment
func LandingPage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello world"))
}

//RefreshToken refreshes authentication token WORK IN PROGRESS not even close to complete
func RefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshSession, _ := sessionStore.Get(r, "refresh-token")

	authSession, _ := sessionStore.Get(r, "auth-token")

	authSession.Values["userID"] = refreshSession.Values["userID"]
	authSession.Options.MaxAge = 60
	authSession.Save(r, w)

}

func HandleFunctions() {
	r := mux.NewRouter()
	r.HandleFunc("/", LandingPage)
	r.HandleFunc("/login", Login).Methods("POST")
	r.HandleFunc("/account", RegisterNewAccount).Methods("POST")
	r.HandleFunc("/account", GetAccountInfo).Methods("GET")
	r.HandleFunc("/resource", ProtectedHandler)
	r.HandleFunc("/events", CreateEvent).Methods("POST")
	r.HandleFunc("/events", GetEvents).Methods("GET")
	r.HandleFunc("/events", JoinEvent).Methods("PUT")
	http.ListenAndServe(":8000", r)
}
