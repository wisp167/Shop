package tests

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wisp167/Shop/internal/server"
)

var app *server.Application

/*
func TestSetupAndTeardown(t *testing.T) {
	app, err := setup()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	defer teardown(app)

	// Add assertions or checks if needed
	t.Log("setup and teardown completed successfully")
}
*/

func setup() (*server.Application, error) {
	app, err := server.SetupApplication()
	if err != nil {
		return nil, err
	}

	// Start the server in a goroutine
	go func() {
		if err := app.Start(); err != nil {
			panic(err)
		}
	}()

	// Wait for the server to start
	time.Sleep(1 * time.Second)

	return app, nil
}

func teardown(app *server.Application) {
	if err := app.Stop(); err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	var err error
	app, err = setup()
	if err != nil {
		panic(err)
	}

	// Run the tests
	code := m.Run()

	// Stop the server
	teardown(app)

	os.Exit(code)
}

// TestAuthAndBuy tests the typical user flow: authentication, buying items, and checking user info.
func TestBuying(t *testing.T) {
	// Step 1: Authenticate a user
	username, password := Generate_Username_Password(1)
	token := authenticateUser(t, username, password)

	// Step 2: Buy an item
	itemName := "t-shirt"
	resp := makeRequest(t, "GET", apiURL+"/buy/"+itemName, token, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Buying an item should return 200 OK")

	// Step 3: Check user info
	coins, inventory := RequestUserInfo(t, token)
	assert.GreaterOrEqual(t, coins, 0, "User coins should be non-negative")
	assert.NotEmpty(t, inventory, "User inventory should not be empty after buying an item")

	// Step 4: Verify the purchased item is in the inventory
	found := false
	for _, item := range inventory {
		if item["item"] == itemName {
			found = true
			break
		}
	}
	assert.True(t, found, "Purchased item should be in the user's inventory")
}

// TestSendCoins tests sending coins between users.
func TestSendCoins(t *testing.T) {
	// Step 1: Authenticate two users
	user1, password1 := Generate_Username_Password(1)
	token1 := authenticateUser(t, user1, password1)

	user2, password2 := Generate_Username_Password(2)
	token2 := authenticateUser(t, user2, password2)

	// Step 2: Send coins from user1 to user2
	payload := fmt.Sprintf(`{"amount": 100, "toUser": "%s"}`, user2)
	resp := makeRequest(t, "POST", apiURL+"/sendCoin", token1, []byte(payload))
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Sending coins should return 200 OK")

	// Step 3: Check user1's balance after sending coins
	coins1, _ := RequestUserInfo(t, token1)
	assert.GreaterOrEqual(t, coins1, 0, "User1's balance should be non-negative after sending coins")

	// Step 4: Check user2's balance after receiving coins
	coins2, _ := RequestUserInfo(t, token2)
	assert.GreaterOrEqual(t, coins2, 100, "User2's balance should reflect the received coins")
}

// TestInsufficientBalance tests sending coins when the sender has insufficient balance.
func TestInsufficientBalance(t *testing.T) {
	// Step 1: Authenticate two users
	user1, password1 := Generate_Username_Password(1)
	token1 := authenticateUser(t, user1, password1)

	user2, password2 := Generate_Username_Password(2)
	authenticateUser(t, user2, password2)

	// Step 2: Check user1's balance
	coins1, _ := RequestUserInfo(t, token1)

	// Step 3: Attempt to send more coins than user1 has
	payload := fmt.Sprintf(`{"amount": %d, "toUser": "%s"}`, coins1+100, user2)
	resp := makeRequest(t, "POST", apiURL+"/sendCoin", token1, []byte(payload))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Sending more coins than balance should return 400 Bad Request")
}

// TestBuyItemWithInsufficientBalance tests buying an item when the user has insufficient balance.
func TestBuyItemWithInsufficientBalance(t *testing.T) {
	username, password := Generate_Username_Password(1)
	token := authenticateUser(t, username, password)

	// Step 2: Check user's balance
	RequestUserInfo(t, token)

	// Step 3: Attempt to buy an item that costs more than the user's balance
	itemName := "pink-hoody" // Assuming this is the most expensive item
	makeRequest(t, "GET", apiURL+"/buy/"+itemName, token, nil)
	makeRequest(t, "GET", apiURL+"/buy/"+"socks", token, nil)

	resp := makeRequest(t, "GET", apiURL+"/buy/"+itemName, token, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Buying an item with insufficient balance should return 400 Bad Request")
}

// TestInvalidAuth tests authentication with invalid credentials.
func TestInvalidAuth(t *testing.T) {
	// Step 1: Attempt to authenticate with invalid credentials
	username, password := Generate_Username_Password(1)
	authenticateUser(t, username, password)
	payload := fmt.Sprintf(`{"username": "%s", "password": "invalid_password"}`, username)
	req, err := http.NewRequest("POST", apiURL+"/auth", bytes.NewBufferString(payload))
	assert.NoError(t, err, "Failed to create auth request")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err, "Auth request failed")
	defer resp.Body.Close()

	// Step 2: Verify the response status code
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Authentication with invalid credentials should return 401 Unauthorized")
}

// TestInvalidBuyRequest tests buying an item with an invalid item name.
func TestInvalidBuyRequest(t *testing.T) {
	// Step 1: Authenticate a user
	username, password := Generate_Username_Password(1)
	token := authenticateUser(t, username, password)

	// Step 2: Attempt to buy an invalid item
	itemName := "invalid-item"
	resp := makeRequest(t, "GET", apiURL+"/buy/"+itemName, token, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Buying an invalid item should return 400 Bad Request")
}

// TestInvalidSendCoinRequest tests sending coins with an invalid request payload.
func TestInvalidSendCoinRequest(t *testing.T) {
	// Step 1: Authenticate a user
	username, password := Generate_Username_Password(1)
	token := authenticateUser(t, username, password)

	// Step 2: Attempt to send coins with an invalid payload
	payload := `{"amount": -100, "toUser": "nonexistent_user"}`
	resp := makeRequest(t, "POST", apiURL+"/sendCoin", token, []byte(payload))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Sending coins with an invalid payload should return 400 Bad Request")
}

// TestUnauthorizedAccess tests accessing protected endpoints without a valid token.
func TestUnauthorizedAccess(t *testing.T) {
	// Step 1: Attempt to access a protected endpoint without a token
	resp := makeRequest(t, "GET", apiURL+"/info", "", nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Accessing a protected endpoint without a token should return 401 Unauthorized")
}

// TestInvalidToken tests accessing protected endpoints with an invalid token.
func TestInvalidToken(t *testing.T) {
	// Step 1: Attempt to access a protected endpoint with an invalid token
	invalidToken := "invalid_token"
	resp := makeRequest(t, "GET", apiURL+"/info", invalidToken, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Accessing a protected endpoint with an invalid token should return 401 Unauthorized")
}
