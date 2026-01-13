package daraja

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MpesaConfig holds our keys
type MpesaConfig struct {
	ConsumerKey    string
	ConsumerSecret string
	BusinessCode   string // The Paybill (174379)
	Passkey        string
	CallbackURL    string // Where Safaricom sends the result
}

// Service is the client we use in our app
type Service struct {
	Config MpesaConfig
	Client *http.Client
}

// NewService creates a new instance
func NewService(key, secret string) *Service {
	return &Service{
		Config: MpesaConfig{
			ConsumerKey:    key,
			ConsumerSecret: secret,
			BusinessCode:   "174379",                                                           // Default Sandbox Paybill
			Passkey:        "bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada1ed2c919", // Default Sandbox Passkey
			CallbackURL:    "https://e3d42c404fab.ngrok-free.app/api/callback",                 // Needs a real URL for callbacks to work
		},
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

// 1. Get Access Token (Auth)
func (s *Service) GetAccessToken() (string, error) {
	url := "https://sandbox.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Basic Auth: Base64(Key:Secret)
	auth := s.Config.ConsumerKey + ":" + s.Config.ConsumerSecret
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Add("Authorization", "Basic "+encodedAuth)

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth failed: %s", string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}

// 2. Trigger STK Push
func (s *Service) InitiateSTKPush(phoneNumber string, amount float64, orderID int) error {
	token, err := s.GetAccessToken()
	if err != nil {
		return err
	}

	url := "https://sandbox.safaricom.co.ke/mpesa/stkpush/v1/processrequest"
	timestamp := time.Now().Format("20060102150405")

	// Password = Base64(Shortcode + Passkey + Timestamp)
	passwordData := s.Config.BusinessCode + s.Config.Passkey + timestamp
	password := base64.StdEncoding.EncodeToString([]byte(passwordData))

	// JSON Payload
	payload := map[string]interface{}{
		"BusinessShortCode": s.Config.BusinessCode,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   "CustomerPayBillOnline",
		"Amount":            int(amount), // Sandbox needs whole numbers
		"PartyA":            phoneNumber, // The customer phone
		"PartyB":            s.Config.BusinessCode,
		"PhoneNumber":       phoneNumber,
		"CallBackURL":       s.Config.CallbackURL,
		"AccountReference":  fmt.Sprintf("Order-%d", orderID),
		"TransactionDesc":   "Payment for Cake",
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("STK Push Failed: %s", string(bodyBytes))
	}

	// Success!
	return nil
}
