package handlers

import (
	"encoding/base64"
	"net/http"
	"strings"
)

func CheckAuth(username, password string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		expectedHeader := getBaseEncodedUserPlusPass(username, password)
		authHeader := parseAuthHeader(req)
		if expectedHeader != authHeader {
			http.Error(res, "Not Authorized", http.StatusUnauthorized)
		}
	}
}

func getBaseEncodedUserPlusPass(username, password string) string {
	data := []byte(username + ":" + password)
	return "basic " + base64.StdEncoding.EncodeToString(data)
}

func parseAuthHeader(req *http.Request) string {
	authString := req.Header.Get("Authorization")

	authHeaderparts := strings.Split(authString, " ")

	if len(authHeaderparts) != 2 {
		return ""
	}

	if strings.ToLower(authHeaderparts[0]) != "basic" {
		return ""
	}

	return "basic " + authHeaderparts[1]
}
