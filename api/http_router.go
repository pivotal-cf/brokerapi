package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

type HttpRouter struct {
	router *mux.Router
}

func NewHttpRouter() HttpRouter {
	return HttpRouter{
		router: mux.NewRouter(),
	}
}

func (httpRouter HttpRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	httpRouter.router.ServeHTTP(w, req)
}

func (httpRouter HttpRouter) Get(url string, handler http.HandlerFunc) {
	httpRouter.router.HandleFunc(url, handler).Methods("GET")
}

func (httpRouter HttpRouter) Put(url string, handler http.HandlerFunc) {
	httpRouter.router.HandleFunc(url, handler).Methods("PUT")
}

func (httpRouter HttpRouter) Delete(url string, handler http.HandlerFunc) {
	httpRouter.router.HandleFunc(url, handler).Methods("DELETE")
}

func (HttpRouter) Vars(req *http.Request) map[string]string {
	return mux.Vars(req)
}
