package dbController

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

func (dbCtrl *DatabaseController) getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	var response JSONResponse

	ctx, canc := context.WithTimeout(context.Background(), time.Second*10)
	defer canc()

	users, err := dbCtrl.DB.GetAllBindings(ctx)
	if err != nil {
		log.Printf("GetAllBindings finished by error: %s\n", err)
		_ = dbCtrl.errorJSON(w, err, http.StatusInternalServerError)
		return
	}

	response.Error = false
	response.Message = "The query was successfully completed"
	response.Data = users

	_ = dbCtrl.writeJSON(w, http.StatusOK, response)
}

func (dbCtrl *DatabaseController) getUserByUserIDHandler(w http.ResponseWriter, r *http.Request) {
	var response JSONResponse

	userID, err := strconv.ParseInt(chi.URLParam(r, "UserID"), 10, 64)
	if err != nil {
		log.Printf("An error occurred in the ParseInt method: %s\n", err)
		_ = dbCtrl.errorJSON(w, err)
		return
	}

	ctx, canc := context.WithTimeout(context.Background(), time.Second*10)
	defer canc()

	users, err := dbCtrl.DB.GetAllBindingsByUserID(ctx, userID)
	if err != nil {
		log.Printf("GetAllBindingsByUserID finished by error: %s\n", err)
		_ = dbCtrl.errorJSON(w, err, http.StatusInternalServerError)
		return
	}

	response.Error = false
	response.Message = "The query was successfully completed"
	response.Data = users

	_ = dbCtrl.writeJSON(w, http.StatusOK, response)
}

func (dbCtrl *DatabaseController) getAllTokensHandler(w http.ResponseWriter, r *http.Request) {
	var response JSONResponse

	ctx, canc := context.WithTimeout(context.Background(), time.Second*10)
	defer canc()

	tokens, err := dbCtrl.DB.GetAllTokens(ctx)
	if err != nil {
		log.Printf("GetAllTokens finished by error: %s\n", err)
		_ = dbCtrl.errorJSON(w, err, http.StatusInternalServerError)
		return
	}

	response.Error = false
	response.Message = "The query was successfully completed"
	response.Data = tokens

	_ = dbCtrl.writeJSON(w, http.StatusOK, response)
}

func (dbCtrl *DatabaseController) setAdminRoleHandler(w http.ResponseWriter, r *http.Request) {
	var entry DatabaseQuerySetAdmin

	err := dbCtrl.readJSON(w, r, &entry)
	if err != nil {
		log.Printf("Failed to read the json request: %s\n", err)
		_ = dbCtrl.errorJSON(w, err)
		return
	}

	ctx, canc := context.WithTimeout(context.Background(), time.Second*10)
	defer canc()

	err = dbCtrl.DB.SetAdminRole(ctx, entry.UserID, entry.State)
	if err != nil {
		log.Printf("An error occurred in SetAdminRole method: %s\n", err)
		_ = dbCtrl.errorJSON(w, err)
		return
	}

	var response JSONResponse

	response.Error = false
	response.Message = fmt.Sprintf("User admin role by %d ID was successfully changed to %t", entry.UserID, entry.State)

	_ = dbCtrl.writeJSON(w, http.StatusOK, response)
}
