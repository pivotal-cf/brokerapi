package auth

import (
	"encoding/base64"
	"net/http"
	"strings"
)

type Wrapper struct {
	username string
	password string
}

func NewWrapper(username, password string) *Wrapper {
	return &Wrapper{
		username: username,
		password: password,
	}
}

const notAuthorized = "Not Authorized"

func (wrapper *Wrapper) Wrap(wrapHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wrapper.username == "" || wrapper.password == "" {
			http.Error(w, notAuthorized, http.StatusUnauthorized)
			return
		}

		expectedHeader := getBaseEncodedUserPlusPass(wrapper.username, wrapper.password)
		authHeader := parseAuthHeader(r)

		if expectedHeader != authHeader {
			http.Error(w, notAuthorized, http.StatusUnauthorized)
			return
		}

		wrapHandler.ServeHTTP(w, r)
	})
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
