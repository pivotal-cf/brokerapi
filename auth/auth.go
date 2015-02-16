package auth

import "net/http"

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
		username, password, isOk := r.BasicAuth()

		if !isOk || username != wrapper.username || password != wrapper.password {
			http.Error(w, notAuthorized, http.StatusUnauthorized)
			return
		}

		wrapHandler.ServeHTTP(w, r)
	})
}
