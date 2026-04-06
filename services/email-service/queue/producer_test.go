package queue

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivationMessage_JSON(t *testing.T) {
	orig := ActivationMessage{
		Email:          "user@example.com",
		FirstName:      "John",
		ActivationLink: "http://example.com/activate/abc123",
	}
	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var decoded ActivationMessage
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, orig, decoded)
}

func TestActivationMessage_JSONKeys(t *testing.T) {
	msg := ActivationMessage{Email: "a@b.com", FirstName: "X", ActivationLink: "http://link"}
	data, _ := json.Marshal(msg)

	var raw map[string]string
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Contains(t, raw, "email")
	assert.Contains(t, raw, "first_name")
	assert.Contains(t, raw, "activation_link")
}

func TestPasswordResetMessage_JSON(t *testing.T) {
	orig := PasswordResetMessage{
		Email:     "user@example.com",
		FirstName: "Jane",
		ResetLink: "http://example.com/reset/xyz",
	}
	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var decoded PasswordResetMessage
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, orig, decoded)
}

func TestPasswordResetMessage_JSONKeys(t *testing.T) {
	msg := PasswordResetMessage{Email: "a@b.com", FirstName: "X", ResetLink: "http://link"}
	data, _ := json.Marshal(msg)

	var raw map[string]string
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Contains(t, raw, "email")
	assert.Contains(t, raw, "first_name")
	assert.Contains(t, raw, "reset_link")
}

func TestPasswordConfirmationMessage_JSON(t *testing.T) {
	orig := PasswordConfirmationMessage{
		Email:     "user@example.com",
		FirstName: "Bob",
	}
	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var decoded PasswordConfirmationMessage
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, orig, decoded)
}

func TestAccountCreatedMessage_JSON(t *testing.T) {
	orig := AccountCreatedMessage{
		Email:         "user@example.com",
		FirstName:     "Ana",
		AccountName:   "Tekući račun",
		AccountNumber: "265000191399797801",
		CurrencyCode:  "RSD",
	}
	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var decoded AccountCreatedMessage
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, orig, decoded)
}

func TestAccountCreatedMessage_JSONKeys(t *testing.T) {
	msg := AccountCreatedMessage{Email: "a@b.com", FirstName: "X", AccountName: "N", AccountNumber: "123", CurrencyCode: "RSD"}
	data, _ := json.Marshal(msg)

	var raw map[string]string
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Contains(t, raw, "email")
	assert.Contains(t, raw, "first_name")
	assert.Contains(t, raw, "account_name")
	assert.Contains(t, raw, "account_number")
	assert.Contains(t, raw, "currency_code")
}

func TestCardConfirmationMessage_JSON(t *testing.T) {
	orig := CardConfirmationMessage{
		Email:            "user@example.com",
		FirstName:        "Marko",
		ConfirmationCode: "482019",
	}
	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var decoded CardConfirmationMessage
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, orig, decoded)
}

func TestCardConfirmationMessage_JSONKeys(t *testing.T) {
	msg := CardConfirmationMessage{Email: "a@b.com", FirstName: "X", ConfirmationCode: "123456"}
	data, _ := json.Marshal(msg)

	var raw map[string]string
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Contains(t, raw, "email")
	assert.Contains(t, raw, "first_name")
	assert.Contains(t, raw, "confirmation_code")
}

func TestLoanLatePaymentMessage_JSON(t *testing.T) {
	orig := LoanLatePaymentMessage{
		Email:      "user@example.com",
		FirstName:  "Petar",
		LoanNumber: "1234567890123",
		AmountDue:  12500.50,
		Currency:   "RSD",
		RetryCount: 2,
	}
	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var decoded LoanLatePaymentMessage
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, orig, decoded)
}

func TestLoanLatePaymentMessage_JSONKeys(t *testing.T) {
	msg := LoanLatePaymentMessage{Email: "a@b.com", FirstName: "X", LoanNumber: "123", AmountDue: 100.0, Currency: "RSD", RetryCount: 1}
	data, _ := json.Marshal(msg)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Contains(t, raw, "email")
	assert.Contains(t, raw, "first_name")
	assert.Contains(t, raw, "loan_number")
	assert.Contains(t, raw, "amount_due")
	assert.Contains(t, raw, "currency")
	assert.Contains(t, raw, "retry_count")
}
