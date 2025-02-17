package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"math/rand"

	"github.com/stretchr/testify/assert"
)

const (
	numUsers           = 100
	apiURL             = "http://localhost:8080/api"
	amountconst        = 1000
	numberofoperations = 10
)

var (
	items  = [10]string{"t-shirt", "cup", "book", "pen", "powerbank", "hoody", "umbrella", "socks", "wallet", "pink-hoody"}
	prices = [10]int{80, 20, 50, 10, 200, 300, 200, 10, 50, 500}
)

func authenticateUser(t *testing.T, username, password string) string {
	payload := fmt.Sprintf(`{"username": "%s", "password": "%s"}`, username, password)
	req, err := http.NewRequest("POST", apiURL+"/auth", bytes.NewBufferString(payload))
	assert.NoError(t, err, "Failed to create auth request")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err, "Auth request failed")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Auth should return 200 OK")

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err, "Failed to decode auth response")

	jwtToken, ok := response["token"].(string)
	assert.True(t, ok, "JWT token not found in auth response")
	assert.NotEmpty(t, jwtToken, "JWT token should not be empty")

	return jwtToken
}

func makeRequest(t *testing.T, method, url, token string, body []byte) *http.Response {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	assert.NoError(t, err, "Failed to create request")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if method == "POST" || method == "PUT" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err, "Request failed")
	return resp
}

func GenerateRandomStringSample(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if length <= 0 {
		return ""
	}

	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	var result strings.Builder

	for i := 0; i < length; i++ {
		randomIndex := seededRand.Intn(len(charset))
		result.WriteByte(charset[randomIndex])
	}

	return result.String()
}

func Generate_Username_Password(i int) (string, string) {
	prefix := GenerateRandomStringSample(6)
	username := prefix + "_" + strconv.Itoa(i)
	password := prefix
	return username, password
}
