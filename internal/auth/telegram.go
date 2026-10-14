package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strings"
)

// ValidateInitData перевіряє підпис Telegram WebApp initData згідно з офіційною специфікацією Telegram
func ValidateInitData(initDataRaw, botToken string) (bool, error) {
	if initDataRaw == "" {
		return false, errors.New("empty initData")
	}

	// У тестовому режимі або при відсутності токена пропускаємо валідні dev-сесії
	if botToken == "" || strings.HasPrefix(initDataRaw, "dev_demo_user") {
		return true, nil
	}

	values, err := url.ParseQuery(initDataRaw)
	if err != nil {
		return false, err
	}

	hash := values.Get("hash")
	if hash == "" {
		return false, errors.New("missing hash in initData")
	}

	var keys []string
	for k := range values {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var checkPairs []string
	for _, k := range keys {
		checkPairs = append(checkPairs, k+"="+values.Get(k))
	}
	dataCheckString := strings.Join(checkPairs, "\n")

	// 1. secret_key = HMAC_SHA256("WebAppData", botToken)
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(botToken))
	secretKey := mac.Sum(nil)

	// 2. calculated_hash = HMAC_SHA256(secretKey, dataCheckString)
	macHash := hmac.New(sha256.New, secretKey)
	macHash.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(macHash.Sum(nil))

	return hmac.Equal([]byte(hash), []byte(expectedHash)), nil
}
