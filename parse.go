package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/metacoin/foundation"
)

var ()

const ROOT_KEY = "alexandria-media"

type AlexandriaMedia struct {
	AlexandriaMedia struct {
		Checksum string `json:"checksum"`
		Info     struct {
			Description string `json:"description"`
			Title       string `json:"title"`
		} `json:"info"`
		Payment struct {
			Amount int64  `json:"amount"`
			Type   string `json:"type"`
		} `json:"payment"`
		Publisher string `json:"publisher"`
		Timestamp int64  `json:"timestamp"`
		Runtime   int64  `json:"runtime"`
		Type      string `json:"type"`
	} `json:"alexandria-media"`
	Signature string `json:"signature"`
}

type AlexandriaPublisher struct {
	AlexandriaPublisher struct {
		Name      string `json:"name"`
		Address   string `json:"address"`
		Timestamp int64  `json:"timestamp"`
	}
	Signature string `json:"signature"`
}

func VerifyMedia(b []byte) (AlexandriaMedia, error) {

	// verify AlexandriaMedia struct
	var v AlexandriaMedia
	var i interface{}
	var m map[string]interface{}

	err := json.Unmarshal(b, &v)
	if err != nil {
		return v, err
	}

	errr := json.Unmarshal(b, &i)
	if errr != nil {
		return v, err
	}

	m = i.(map[string]interface{})
	var signature string

	// check the JSON object root key
	// find the signature string
	for key, val := range m {
		if key == "signature" {
			signature = val.(string)
		} else {
			if key != ROOT_KEY {
				return v, errors.New("JSON object root key doesn't match accepted value")
			}
		}
	}

	// verify checksum length
	if len(v.AlexandriaMedia.Checksum) <= 1 {
		return v, errors.New("invalid checksum length")
	}

	// verify signature
	if v.Signature != signature {
		return v, errors.New("signature mismatch")
	}

	// verify signature was created by this address
	address := v.AlexandriaMedia.Publisher
	checksum := v.AlexandriaMedia.Checksum
	timestamp := strconv.FormatInt(v.AlexandriaMedia.Timestamp, 10)

	if foundation.RPCCall("verifymessage", address, signature, checksum+timestamp) != true {
		return v, errors.New("message failed to pass signature verification")
	}

	return v, nil

}

func Store() {

}

func main() {
	data := []byte(`{ "alexandria-media": { "checksum": "sha256", "publisher": "FFbtpjAUQdNVnHyKyFLHYTxG5bX5PxcUAp", "timestamp": 12345, "type": "song", "payment": { "type": "FLO", "amount": 1 }, "runtime": 130, "info": { "title": "A Song Title", "description": "Description!" } }, "signature":"H+kObAOMNX/YiD06uVrLZjDFdgU3HOL013iKORBtRfrQF0F3e1yPxARCAAxxf8kscx64811cunBs3YRt+OtKY3I=" }`)

	am, err := VerifyMedia(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n\n%v", am)

}
