package tests

import (
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
			token := authenticateUser(t, username, password)

			var chosen_item int
			amount := amountconst
			for j := 0; j < numberofoperations; j++ {
				if j%5 != userID%5 {
					chosen_item = 9
				} else {
					chosen_item = rand.Intn(10)
				}
				resp := makeRequest(t, "GET", apiURL+"/buy/"+items[chosen_item], token, nil)
				if prices[chosen_item] > amount && resp.StatusCode != http.StatusBadRequest {
					t.Errorf("User %d: Didn't receive 400 status when he should've: %d\n", userID, resp.StatusCode)
					return
				} else if prices[chosen_item] <= amount && resp.StatusCode != http.StatusOK {
					t.Errorf("User %d: Received non-200 status: %d\n", userID, resp.StatusCode)
					return
				}
				if prices[chosen_item] <= amount {
					amount -= prices[chosen_item]
				}
			}

			t.Logf("User %d: Request successful\n", userID)
		}(i)
	}

	wg.Wait()

	t.Logf("Load test completed in %v\n", time.Since(startTime))
	t.Logf("average response time: %v\n", time.Since(startTime)/(time.Duration(numUsers)*numberofoperations))
}
