package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"math/rand"
)

func TestAuthAndBuy(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(numUsers)
	rand.Seed(time.Now().UnixNano()) // Use current time as seed)

	startTime := time.Now()

	for i := 0; i < numUsers; i++ {
		go func(userID int) {
			defer wg.Done()
			username, password := Generate_Username_Password(userID)
			// Prepare the request payload
			payload := fmt.Sprintf(`{"username": "%s", "password": "%s"}`, username, password)
			req, err := http.NewRequest("POST", apiURL+"/auth", bytes.NewBufferString(payload))
			if err != nil {
				t.Errorf("User %d: Failed to create request: %v\n", userID, err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("User %d: Request failed: %v\n", userID, err)
				return
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("User %d: Received non-200 status during auth: %d\n", userID, resp.StatusCode)
				return
			}
			var response map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Errorf("User %d: Failed to decode auth response: %v\n", userID, err)
				return
			}
			jwtToken := response["token"].(string)
			resp.Body.Close()
			if jwtToken == "" {
				t.Errorf("User %d: JWT token not found in auth response\n", userID)
			}
			var chosen_item int
			amount := amountconst
			for j := 0; j < numberofoperations; j++ {
				if j%5 != userID%5 {
					chosen_item = 9
				} else {
					chosen_item = rand.Intn(10)
				}
				req, err = http.NewRequest("GET", apiURL+"/buy/"+items[chosen_item], nil)
				req.Header.Set("Authorization", "Bearer "+jwtToken)
				if err != nil {
					t.Errorf("User %d: Failed to create request: %v\n", userID, err)
					return
				}
				resp, err = client.Do(req)
				if err != nil {
					t.Errorf("User %d: Request failed: %v\n", userID, err)
					return
				}
				if prices[chosen_item] > amount && resp.StatusCode != http.StatusBadRequest {
					t.Errorf("User %d: Didn't receive 400 status when he should've: %d\n", userID, resp.StatusCode)
					return
				} else if prices[chosen_item] <= amount && resp.StatusCode != http.StatusOK {
					t.Errorf("User %d: Received non-200 status: %d\n", userID, resp.StatusCode)
					return
				}
				resp.Body.Close()
				if prices[chosen_item] <= amount {
					amount -= prices[chosen_item]
				}
			}
			/*
				req.Header.Set("Authorization", "Bearer "+jwtToken)
				defer resp.Body.Close()

				// Check the response status
				if resp.StatusCode != http.StatusOK {
					fmt.Printf("User %d: Received non-200 status: %d\n", userID, resp.StatusCode)
					return
				}
			*/

			t.Logf("User %d: Request successful\n", userID)
		}(i)
	}

	wg.Wait()

	t.Logf("Load test completed in %v\n", time.Since(startTime))
	t.Logf("average response time: %v\n", time.Since(startTime)/(time.Duration(numUsers)*numberofoperations))
}
