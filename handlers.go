package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

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
	r.HandleFunc("/login", IsLoggedIn).Methods("GET")
	r.HandleFunc("/login", Login).Methods("POST")
	r.HandleFunc("/login", Logout).Methods("DELETE")

	r.HandleFunc("/account", RegisterNewAccount).Methods("POST")
	r.HandleFunc("/account", GetAccountInfo).Methods("GET")
	r.HandleFunc("/account", EditAccountInfo).Methods("PATCH")

	r.HandleFunc("/events", GetEvents).Methods("GET")

	r.HandleFunc("/events", CreateEvent).Methods("POST")
	r.HandleFunc("/events/{id}", EditEvent).Methods("PATCH")
	r.HandleFunc("/events/{id}", DeleteEvent).Methods("DELETE")

	r.HandleFunc("/events/{id}/users", JoinEvent).Methods("PATCH")
	r.HandleFunc("/events/{id}/users", LeaveEvent).Methods("DELETE")
	http.ListenAndServe(":8000", r)
}
