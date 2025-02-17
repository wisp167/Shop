package server

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

func (app *Application) logError(r *http.Request, err error) {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}
	_, file1, line1, ok1 := runtime.Caller(3)
	if !ok1 {
		file1 = "unknown"
		line = 0
	}

	// Extract just the filename from the full path
	fileParts := strings.Split(file, "/")
	filename := fileParts[len(fileParts)-1]
	fileParts = strings.Split(file1, "/")
	filename1 := fileParts[len(fileParts)-1]
	// Log the error with the file and line number
	app.logger.Printf("[%s:%d]->[%s:%d] %v", filename, line, filename1, line1, err)
}

func (app *Application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	env := envelope{"error": message}
	err := app.writeJSON(w, status, env, nil)
	if err != nil {
		app.logError(r, err)
	}
}

func (app *Application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)

	message := "Внутренняя ошибка сервера."
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (app *Application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

func (app *Application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

func (app *Application) badRequestResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("Неверный запрос.")
	app.errorResponse(w, r, http.StatusBadRequest, message)
}

func (app *Application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

func (app *Application) authorizationErrorResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("Неавторизован.")
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}
