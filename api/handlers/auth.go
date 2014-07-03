package handlers

import (
	"encoding/base64"
	"net/http"
	"os"
	"strings"
)

func CheckAuth() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		envString := getBaseEncodedUserPlusPass()
		authHeader := parseAuthHeader(req)
		if envString != authHeader {
			http.Error(res, "Not Authorized", http.StatusUnauthorized)
		}
	}
}

func getBaseEncodedUserPlusPass() string {
	username := os.Getenv("BROKER_USER")
	password := os.Getenv("BROKER_PASSWORD")
	data := []byte(username + ":" + password)
	return "basic " + base64.StdEncoding.EncodeToString(data)
}

func parseAuthHeader(req *http.Request) string {
	authString := req.Header.Get("Authorization")
	encryptedKey := strings.Split(authString, " ")[1]
	return "basic " + encryptedKey
}
