package main

import (
	"math/rand"
	"time"
)

const (
	alphabet  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	keylength = 32
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateKey() (key string) {
	for len(key) < keylength {
		i := rand.Intn(len(alphabet))
		key += string(alphabet[i])
	}
	return
}

// User represents a user in table "apikey".
type User struct {
	ID       uint      `json:"-"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Key      string    `json:"apikey"`
	Note     string    `json:"note"`
}

// Pair represents a key-value pair in table "kvpair".
type Pair struct {
	ID       uint      `json:"-"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	OwnerID  uint      `json:"-" db:"owner_id"` // FK to apikey (User)
	Key      string    `json:"key"`
	Value    string    `json:"value"`
}
