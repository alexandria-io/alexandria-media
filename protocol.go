package alexandriaMedia

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/metacoin/foundation"
)

var ()

const MEDIA_ROOT_KEY = "alexandria-media"
const PUBLISHER_ROOT_KEY = "alexandria-publisher"
const MIN_BLOCK = 984588

type AlexandriaMedia struct {
	AlexandriaMedia struct {
		Torrent   string `json:"torrent"`
		Publisher string `json:"publisher"`
		Timestamp int64  `json:"timestamp"`
		Runtime   int64  `json:"runtime"`
		Info      struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"info"`
		Payment struct {
			Amount int64  `json:"amount"`
			Type   string `json:"type"`
		} `json:"payment"`
		Extras string `json:"extras"`
		Type   string `json:"type"`
	} `json:"alexandria-media"`
	Signature string `json:"signature"`
}

type AlexandriaPublisher struct {
	AlexandriaPublisher struct {
		Name      string `json:"name"`
		Address   string `json:"address"`
		Timestamp int64  `json:"timestamp"`
	} `json:"alexandria-publisher"`
	Signature string `json:"signature"`
}

// reference: Cory LaNou, Mar 2 '14 at 15:21, http://stackoverflow.com/a/22129435/2576956
func isJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func VerifyPublisher(b []byte) (AlexandriaPublisher, error) {

	var v AlexandriaPublisher
	var i interface{}
	var m map[string]interface{}

	// fmt.Printf("Attempting to verify alexandria-publisher JSON...")

	if !isJSON(string(b)) {
		return v, errors.New("this string isn't even JSON!")
	}

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
			if key != PUBLISHER_ROOT_KEY {
				return v, errors.New("can't verify publisher - JSON object root key doesn't match accepted value")
			}
		}
	}

	// verify signature
	if v.Signature != signature {
		return v, errors.New("can't verify publisher - signature mismatch")
	}

	// verify signature was created by this address
	// signature pre-image for publisher is the string concatenation of name+address+timestamp
	if checkSignature(v.AlexandriaPublisher.Address, signature, v.AlexandriaPublisher.Name+v.AlexandriaPublisher.Address+strconv.FormatInt(v.AlexandriaPublisher.Timestamp, 10)) == false {
		return v, errors.New("can't verify publisher - message failed to pass signature verification")
	}

	// fmt.Println(" -- VERIFIED --")
	return v, nil

}

func VerifyMedia(b []byte) (AlexandriaMedia, error) {

	var v AlexandriaMedia
	var i interface{}
	var m map[string]interface{}

	// fmt.Printf("Attempting to verify alexandria-media JSON...")
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
			if key != MEDIA_ROOT_KEY {
				return v, errors.New("can't verify media - JSON object root key doesn't match accepted value")
			}
		}
	}

	// verify torrent hash length
	if len(v.AlexandriaMedia.Torrent) <= 1 {
		return v, errors.New("can't verify media - invalid checksum length")
	}

	// verify signature
	if v.Signature != signature {
		return v, errors.New("can't verify media - signature mismatch")
	}

	// verify signature was created by this address
	// signature pre-image for media is the string concatenation of torrenthash+timestamp
	if checkSignature(v.AlexandriaMedia.Publisher, signature, v.AlexandriaMedia.Torrent+strconv.FormatInt(v.AlexandriaMedia.Timestamp, 10)) == false {
		return v, errors.New("can't verify media - message failed to pass signature verification")
	}

	// fmt.Println(" -- VERIFIED --")
	return v, nil

}

func StorePublisher(publisher AlexandriaPublisher, dbtx *sql.Tx, txid string, block int, hash string) {
	// store in database
	stmtstr := `insert into publisher (name, address, timestamp, txid, block, hash, signature, active) values (?, ?, ?, "` + txid + `", ` + strconv.Itoa(block) + `, "` + hash + `", ?, 1)`

	stmt, err := dbtx.Prepare(stmtstr)
	if err != nil {
		fmt.Println("exit 100")
		log.Fatal(err)
	}

	_, stmterr := stmt.Exec(publisher.AlexandriaPublisher.Name, publisher.AlexandriaPublisher.Address, publisher.AlexandriaPublisher.Timestamp, publisher.Signature)
	if err != nil {
		fmt.Println("exit 101")
		log.Fatal(stmterr)
	}

	stmt.Close()

}

func checkSignature(address string, signature string, message string) bool {
	if foundation.RPCCall("verifymessage", address, signature, message) == true {
		return true
	}
	return false
}

/*
func main() {
		data := []byte(`{ "alexandria-media": { "checksum": "sha256", "publisher": "FFbtpjAUQdNVnHyKyFLHYTxG5bX5PxcUAp", "timestamp": 12345, "type": "song", "payment": { "type": "FLO", "amount": 1 }, "runtime": 130, "info": { "title": "A Song Title", "description": "Description!" } }, "signature":"H+kObAOMNX/YiD06uVrLZjDFdgU3HOL013iKORBtRfrQF0F3e1yPxARCAAxxf8kscx64811cunBs3YRt+OtKY3I=" }`)

		pdata := []byte(`{ "alexandria-publisher": { "name": "Joey", "address": "FFbtpjAUQdNVnHyKyFLHYTxG5bX5PxcUAp", "timestamp": 12345 }, "signature":"IJ+YGrBzqIxPaoUFm3959/ucZcMZn/DURDFyFq7dRH5/4arrlCg9ip2jgmqothac+0OiBh1fiSIIESf6lpjJazw="} `)

		_, err := VerifyMedia(data)
		if err != nil {
			log.Fatal(err)
		}

		_, errr := VerifyPublisher(pdata)
		if errr != nil {
			log.Fatal(err)
		}

}
*/
