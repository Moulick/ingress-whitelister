package utils

import (
	"math/rand"
	"slices"

	jsoniter "github.com/json-iterator/go"
)

func ConvertFromJSON(jsonString string) (jsoniter.Any, error) {
	jsonC := jsoniter.Config{
		EscapeHTML:                    false,
		MarshalFloatWith6Digits:       true,
		ObjectFieldMustBeSimpleString: true,
		SortMapKeys:                   true,
		ValidateJsonRawMessage:        true,
	}

	var data jsoniter.Any
	err := jsonC.Froze().Unmarshal([]byte(jsonString), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ArrayInArray returns true if x is a subset of y
func ArrayInArray(x, y []string) (bool, string) {
	if len(x) > len(y) {
		return false, ""
	}

	for _, xVal := range x {
		present := slices.Contains(y, xVal)
		if !present {
			return false, xVal
		}
	}
	return true, ""
}

var letterRunes = []rune(`abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890,./;'[]\<>?:"{}|~!@#$%^&*()_+"'`)

func randomString(n int, seed int64) string {
	rand.New(rand.NewSource(seed))
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
