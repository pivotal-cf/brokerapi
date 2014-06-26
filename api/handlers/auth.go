package handlers

import (
	"encoding/base64"
	"net/http"
	"os"
)

func CheckAuth() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		envString := getBaseEncodedUserPlusPass()
		if envString != req.Header.Get("Authorization") {
			http.Error(res, "Not Authorized", http.StatusUnauthorized)
		}
	}
}

func getBaseEncodedUserPlusPass() string {
	username := os.Getenv("BROKER_USER")
	password := os.Getenv("BROKER_PASSWORD")
	data := []byte(username + ":" + password)
	return "Basic " + base64.StdEncoding.EncodeToString(data)
}
