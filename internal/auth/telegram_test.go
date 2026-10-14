package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const token = "123456:TEST-TOKEN"

var now = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

// sign builds a correctly signed initData for the given fields.
func sign(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+fields[k])
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(token))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(pairs, "\n")))

	v := url.Values{}
	for k, val := range fields {
		v.Set(k, val)
	}
	v.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return v.Encode()
}

func fields(authDate time.Time) map[string]string {
	return map[string]string{
		"auth_date": strconv.FormatInt(authDate.Unix(), 10),
		"user":      `{"id":42,"first_name":"Val"}`,
		"query_id":  "AAH",
	}
}

func TestValidateAcceptsSignedData(t *testing.T) {
	p, err := Validate(sign(fields(now.Add(-time.Hour))), token, now)
	if err != nil || p.ID != "42" {
		t.Fatalf("got %+v, %v", p, err)
	}
}

func TestValidateRejects(t *testing.T) {
	good := sign(fields(now.Add(-time.Minute)))
	tests := []struct {
		name, data, token string
		want              error
	}{
		{"empty", "", token, ErrNoInitData},
		{"wrong token", good, "999:OTHER", ErrBadHash},
		{"no token configured", good, "", ErrBadHash},
		{"no hash", "auth_date=1&user=%7B%22id%22%3A1%7D", token, ErrBadHash},
		{"forged hash", "auth_date=" + strconv.FormatInt(now.Unix(), 10) + `&user=%7B%22id%22%3A1%7D&hash=` + strings.Repeat("0", 64), token, ErrBadHash},
		// The old code let any string starting with "dev_demo_user" through.
		{"dev prefix is not a bypass", "dev_demo_user", token, ErrBadHash},
		{"expired", sign(fields(now.Add(-25 * time.Hour))), token, ErrExpired},
		{"from the future", sign(fields(now.Add(time.Hour))), token, ErrExpired},
	}
	for _, tc := range tests {
		if _, err := Validate(tc.data, tc.token, now); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}
	// A valid signature must not carry over to a changed payload.
	v, _ := url.ParseQuery(good)
	v.Set("user", `{"id":7}`)
	if _, err := Validate(v.Encode(), token, now); !errors.Is(err, ErrBadHash) {
		t.Errorf("tampered user: err = %v, want ErrBadHash", err)
	}
}
