package tests

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestLoadCoinTransferBetweenOddAndEvenUsers(t *testing.T) {
	const (
		numUsers       = 10 // Total number of users
		numIterations  = 10 // Number of transfers per user
		transferAmount = 1  // Amount to transfer each time
	)

	var wg sync.WaitGroup
	wg.Add(numUsers)

	startTime := time.Now()

	// Authenticate all users and store their tokens
	tokens := make([]string, numUsers)
	for i := 0; i < numUsers; i++ {
		temp := fmt.Sprintf("user_%d", i)
		username, password := temp, temp
		tokens[i] = authenticateUser(t, username, password)
	}

	// Simulate transfers between odd and even users
	for i := 0; i < numUsers; i++ {
		go func(userID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				// Determine the target user
				targetUserID := (userID + 1) % numUsers // Transfer to the next user
				if userID%2 == 0 {                      // Even users send to odd users
					targetUserID = (userID + 1) % numUsers
				} else { // Odd users send to even users
					targetUserID = (userID - 1 + numUsers) % numUsers
				}

				// Get the target user's username
				targetUsername := fmt.Sprintf("user_%d", targetUserID)

				// Prepare the request payload
				payload := fmt.Sprintf(`{"amount": %d, "toUser": "%s"}`, transferAmount, targetUsername)
				resp := makeRequest(t, "POST", apiURL+"/sendCoin", tokens[userID], []byte(payload))
				counter := 10
				for counter > 0 && (resp.StatusCode != http.StatusOK) {
					resp = makeRequest(t, "POST", apiURL+"/sendCoin", tokens[userID], []byte(payload))
					counter--
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("User %d: Received non-200 status during transfer: %d\n", userID, resp.StatusCode)
				}
				RequestUserInfo(t, tokens[userID])
				t.Logf("User %d: Received Status Code %d", userID, resp.StatusCode)

			}
		}(i)
	}

	wg.Wait()

	// Log the total time taken
	t.Logf("Load test completed in %v", time.Since(startTime))
	t.Logf("Total transfers: %d", numUsers*numIterations)
	t.Logf("Average response time: %v", time.Since(startTime)/(numUsers*numIterations*2))
}
