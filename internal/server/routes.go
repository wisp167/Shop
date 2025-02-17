package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *Application) routes() *httprouter.Router {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthz", app.healthcheckHandler) //health
	router.HandlerFunc(http.MethodPost, "/api/auth", app.authHandler)
	router.HandlerFunc(http.MethodGet, "/api/buy/:item", app.jwtMiddleware(app.buyItemHandler))
	router.HandlerFunc(http.MethodPost, "/api/sendCoin", app.jwtMiddleware(app.sendCoinHandler))
	router.HandlerFunc(http.MethodGet, "/api/info", app.jwtMiddleware(app.getInfoHandler))
	return router
}
