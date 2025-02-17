package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/wisp167/Shop/internal/data"
)

func (app *Application) buyItemHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	app.queue <- struct{}{}
	defer func() {
		<-app.queue
	}()
	app.buyItemWorker(w, r, ps)
}

func (app *Application) buyItemWorker(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (err error) {
	itemName := ps.ByName("item")
	if itemName == "" {
		app.logger.Println("No item name provided")
		app.badRequestResponse(w, r)
		return nil
	}
	userID, ok := r.Context().Value("id").(int64)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("cannot get user id"))
		return nil
	}
	tx, err := app.models.Shop.DB.BeginTx(context.Background(), nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	item, err := app.models.Shop.GetItemByName(itemName)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}
	if item == nil {
		app.badRequestResponse(w, r)
		err = errors.New("item not found")
		return err
	}
	user, err := app.models.Shop.GetUserByID(userID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}
	if user.Balance < item.Price {
		app.badRequestResponse(w, r)
		err = errors.New("")
		return err
	}
	err = app.models.Shop.UpdateUserBalanceAfterPurchase(tx, userID, item.Price)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		err = errors.New("")
		return err
	}
	err = app.models.Shop.InsertUserItem(tx, userID, item.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}
	app.writeJSON(w, http.StatusOK, envelope{}, nil)
	return nil
}

func (app *Application) sendCoinHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	app.queue <- struct{}{}
	defer func() {
		<-app.queue
	}()
	app.sendCoinWorker(w, r, httprouter.Params{})
}
func (app *Application) sendCoinWorker(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (err error) {
	var request struct {
		Amount   int    `json:"amount"`
		Receiver string `json:"toUser"`
	}

	// Read and parse the JSON request body
	if err := app.readJSON(w, r, &request); err != nil {
		app.logger.Printf("Error reading JSON: %v", err)
		app.badRequestResponse(w, r)
		return err
	}

	// Validate the request
	if request.Amount <= 0 || request.Receiver == "" {
		app.logger.Println("Invalid request: amount or receiver is empty")
		app.badRequestResponse(w, r)
		return errors.New("invalid request")
	}

	// Get the user ID from the context
	userID, ok := r.Context().Value("id").(int64)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("cannot get user id"))
		return errors.New("cannot get user id")
	}

	// Start a database transaction
	tx, err := app.models.Shop.DB.BeginTx(context.Background(), nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Get the sender's details
	sender, err := app.models.Shop.GetUserByID(userID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}

	// Validate sender's balance and receiver
	if sender.Balance < request.Amount || sender.Username == request.Receiver {
		app.badRequestResponse(w, r)
		return errors.New("insufficient balance or invalid receiver")
	}

	// Update sender's balance
	if err := app.models.Shop.UpdateSenderBalance(tx, userID, request.Amount); err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}

	// Get the receiver's details
	receiver, err := app.models.Shop.GetUserByUsername(request.Receiver)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}
	if receiver == nil {
		app.badRequestResponse(w, r)
		return errors.New("receiver not found")
	}

	// Update receiver's balance
	if err := app.models.Shop.UpdateReceiverBalance(tx, receiver.ID, request.Amount); err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}

	// Record the transaction
	if err := app.models.Shop.InsertTransaction(tx, sender.ID, receiver.ID, request.Amount); err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}

	// Return a success response
	app.writeJSON(w, http.StatusOK, envelope{}, nil)
	return nil
}

func (app *Application) getInfoHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	app.queue <- struct{}{}
	defer func() {
		<-app.queue
	}()
	app.getInfoWorker(w, r, ps)
}

func (app *Application) getInfoWorker(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (err error) {
	// Get the user ID from the context
	userID, ok := r.Context().Value("id").(int64)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("cannot get user id"))
		return errors.New("cannot get user id")
	}

	// Fetch user balance and inventory
	balance, inventory, err := app.models.Shop.GetUserBalanceAndInventory(userID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}

	// Fetch transaction history with usernames
	transactions, err := app.models.Shop.GetTransactionHistoryWithUsernames(userID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}

	// Prepare the response
	response := struct {
		Coins       int         `json:"coins"`
		Inventory   []data.Item `json:"inventory"`
		CoinHistory struct {
			Received []struct {
				FromUser string `json:"fromUser"`
				Amount   int    `json:"amount"`
			} `json:"received"`
			Sent []struct {
				ToUser string `json:"toUser"`
				Amount int    `json:"amount"`
			} `json:"sent"`
		} `json:"coinHistory"`
	}{
		Coins:     balance,
		Inventory: inventory,
	}

	// Populate the coin history
	for _, t := range transactions {
		if t.ToUserID == userID {
			// Received transaction
			response.CoinHistory.Received = append(response.CoinHistory.Received, struct {
				FromUser string `json:"fromUser"`
				Amount   int    `json:"amount"`
			}{
				FromUser: t.FromUser,
				Amount:   t.Amount,
			})
		} else if t.FromUserID == userID {
			// Sent transaction
			response.CoinHistory.Sent = append(response.CoinHistory.Sent, struct {
				ToUser string `json:"toUser"`
				Amount int    `json:"amount"`
			}{
				ToUser: t.ToUser,
				Amount: t.Amount,
			})
		}
	}

	// Return the response
	app.writeJSON(w, http.StatusOK, response, nil)
	return nil
}

/*
func (app *Application) getInfoWorker(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (err error) {
	// Get the user ID from the context
	userID, ok := r.Context().Value("id").(int64)
	if !ok {
		app.serverErrorResponse(w, r, errors.New("cannot get user id"))
		return errors.New("cannot get user id")
	}

	// Fetch user balance and inventory
	balance, inventory, err := app.models.Shop.GetUserBalanceAndInventory(userID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}

	// Fetch transaction history
	transactions, err := app.models.Shop.GetTransactionHistory(userID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return err
	}

	// Prepare the response
	response := struct {
		Coins       int         `json:"coins"`
		Inventory   []data.Item `json:"inventory"`
		CoinHistory struct {
			Received []struct {
				FromUser string `json:"fromUser"`
				Amount   int    `json:"amount"`
			} `json:"received"`
			Sent []struct {
				ToUser string `json:"toUser"`
				Amount int    `json:"amount"`
			} `json:"sent"`
		} `json:"coinHistory"`
	}{
		Coins:     balance,
		Inventory: inventory,
	}

	// Populate the coin history
	for _, t := range transactions {
		if t.ToUserID == int(userID) {
			// Received transaction
			fromUser, err := app.models.Shop.GetUserByID(int64(t.FromUserID))
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return err
			}
			response.CoinHistory.Received = append(response.CoinHistory.Received, struct {
				FromUser string `json:"fromUser"`
				Amount   int    `json:"amount"`
			}{
				FromUser: fromUser.Username,
				Amount:   t.Amount,
			})
		} else if t.FromUserID == int(userID) {
			// Sent transaction
			toUser, err := app.models.Shop.GetUserByID(int64(t.ToUserID))
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return err
			}
			response.CoinHistory.Sent = append(response.CoinHistory.Sent, struct {
				ToUser string `json:"toUser"`
				Amount int    `json:"amount"`
			}{
				ToUser: toUser.Username,
				Amount: t.Amount,
			})
		}
	}

	app.writeJSON(w, http.StatusOK, response, nil)
	return nil
}
*/
