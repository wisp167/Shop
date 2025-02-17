package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
)

type Claims struct {
	Userid int64
	jwt.StandardClaims
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

func (app *application) jwtMiddleware(next httprouter.Handle) http.HandlerFunc {
	return wrapHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return app.jwtkey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add the username to the request context
		ctx := context.WithValue(r.Context(), "id", claims.Userid)
		r = r.WithContext(ctx)

		// Call the next handler
		next(w, r, ps)
	})
}

func (app *application) authHandler(w http.ResponseWriter, r *http.Request) { // Modified to enqueue
	app.queue <- struct{}{}
	defer func() {
		<-app.queue
	}()
	app.authWorker(w, r, httprouter.Params{})

}

func (app *application) authWorker(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var req AuthRequest
	if err := app.readJSON(w, r, &req); err != nil {
		app.logger.Printf("Error reading JSON: %v", err)
		app.badRequestResponse(w, r)
		return
	}
	// Validate user credentials (e.g., check against database)
	// For simplicity, we'll assume the credentials are valid
	user, err := app.models.Shop.GetUserByUsername(req.Username)
	if user == nil {
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		user, err = app.models.Shop.InsertUser(req.Username, req.Password)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		user.Password = req.Password
	}
	if user.Password != req.Password {
		app.authorizationErrorResponse(w, r)
		return
	}

	// Create JWT token
	expirationTime := time.Now().Add(time.Hour)
	claims := &Claims{
		Userid: user.ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(app.jwtkey)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return the token
	response := AuthResponse{Token: tokenString}
	app.writeJSON(w, http.StatusOK, response, nil)
}
