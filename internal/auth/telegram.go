package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// MaxAge is how long a signed initData stays acceptable (auth_date + MaxAge).
const MaxAge = 24 * time.Hour

var (
	ErrNoInitData = errors.New("missing initData")
	ErrBadHash    = errors.New("invalid initData signature")
	ErrExpired    = errors.New("initData expired")
)

// Principal is the authenticated caller. Sessions are bound to its ID.
type Principal struct {
	ID string
}

// DevPrincipal is used only when the server runs with DEV_AUTH=1.
var DevPrincipal = Principal{ID: "dev"}

// Validate checks a Telegram WebApp initData string as specified at
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
// and returns the caller. It never accepts an unsigned string: the dev bypass
// lives in the HTTP layer behind an explicit flag, not in client-controlled data.
func Validate(initData, botToken string, now time.Time) (Principal, error) {
	if initData == "" {
		return Principal{}, ErrNoInitData
	}
	if botToken == "" {
		return Principal{}, ErrBadHash
	}
	values, err := url.ParseQuery(initData)
	if err != nil {
		return Principal{}, err
	}
	hash := values.Get("hash")
	if hash == "" {
		return Principal{}, ErrBadHash
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+values.Get(k))
	}

	// secret_key = HMAC_SHA256(key="WebAppData", msg=bot_token)
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	// hash = HMAC_SHA256(key=secret_key, msg=data_check_string)
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(pairs, "\n")))
	want := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(hash), []byte(want)) {
		return Principal{}, ErrBadHash
	}

	sec, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return Principal{}, ErrExpired
	}
	if age := now.Sub(time.Unix(sec, 0)); age > MaxAge || age < -time.Minute {
		return Principal{}, ErrExpired
	}

	var user struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID == 0 {
		return Principal{}, ErrBadHash
	}
	return Principal{ID: strconv.FormatInt(user.ID, 10)}, nil
}
