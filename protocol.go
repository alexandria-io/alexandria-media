package alexandriaMedia

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/metacoin/foundation"
)

var ()

const MEDIA_ROOT_KEY = "alexandria-media"
const PUBLISHER_ROOT_KEY = "alexandria-publisher"
const MIN_BLOCK = 1002555

// media structs
type AlexandriaMedia struct {
	AlexandriaMedia struct {
		Torrent   string `json:"torrent"`
		Publisher string `json:"publisher"`
		Timestamp int64  `json:"timestamp"`
		Info      struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Year        int    `json:"year"`
			Size        int64  `json:"size"`
		} `json:"info"`
		Payment struct {
			Currency string `json:"currency"`
			Type     string `json:"type"`
			Amount   int64  `json:"amount"`
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

// multipart structs
type MediaMultipartSingle struct {
	Part      int
	Max       int
	Reference string
	Data      string
	Txid      string
	Block     int
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
	if checkSignature(v.AlexandriaPublisher.Address, signature, v.AlexandriaPublisher.Name+"-"+v.AlexandriaPublisher.Address+"-"+strconv.FormatInt(v.AlexandriaPublisher.Timestamp, 10)) == false {
		return v, errors.New("can't verify publisher - message failed to pass signature verification")
	}

	// fmt.Println(" -- VERIFIED --")
	return v, nil

}

func StoreMediaMultipartSingle(mms MediaMultipartSingle, dbtx *sql.Tx) {
	// store in database
	stmtstr := `insert into media_multipart (part, max, reference, data, txid, block, complete, active) values (` + strconv.Itoa(mms.Part) + `, ` + strconv.Itoa(mms.Max) + `, "` + mms.Reference + `", ?, "` + mms.Txid + `", ` + strconv.Itoa(mms.Block) + `, 0, 1)`

	stmt, err := dbtx.Prepare(stmtstr)
	if err != nil {
		fmt.Println("exit 100")
		log.Fatal(err)
	}

	_, stmterr := stmt.Exec(mms.Data)
	if err != nil {
		fmt.Println("exit 106")
		log.Fatal(stmterr)
	}

	stmt.Close()

}

func VerifyMediaMultipartSingle(s string, txid string, block int) (MediaMultipartSingle, error) {
	var ret MediaMultipartSingle
	prefix := "alexandria-media-multipart("

	// check prefix
	checkPrefix := strings.HasPrefix(s, prefix)
	if !checkPrefix {
		return ret, errors.New("wrong prefix in tx-comment (does not match required prefix)")
	}

	// trim prefix off
	s = strings.TrimPrefix(s, prefix)

	// check part and max
	part, err := strconv.Atoi(string(s[0]))
	if err != nil {
		fmt.Println("cannot convert part to int")
		return ret, errors.New("cannot convert part to int")
	}
	max, err2 := strconv.Atoi(string(s[2]))
	if err2 != nil {
		fmt.Println("cannot convert max to int")
		return ret, errors.New("cannot convert max to int")
	}

	// check reference
	reference := "error"
	data := ""
	if part == 0 {
		reference = "0"
		ind := strings.Index(s, "):")
		data = s[ind+2:]
	} else {
		// fmt.Printf("# # length check: %v # # # \n", len(s))
		if len(s) < 71 {
			return ret, errors.New("not enough data in mutlipart string")
		}
		reference = s[4:68]
		data = s[70:]
	}

	// fmt.Printf("data: %v\n", data)
	// fmt.Printf("=== VERIFIED ===\n")
	// fmt.Printf("part: %v\nmax: %v\nreference: %v\n", part, max, reference)

	ret = MediaMultipartSingle{
		Part:      part,
		Max:       max,
		Reference: reference,
		Data:      data,
		Txid:      txid,
		Block:     block,
	}

	return ret, nil

}

func VerifyMedia(b []byte) (AlexandriaMedia, map[string]interface{}, error) {

	var v AlexandriaMedia
	var i interface{}
	var m map[string]interface{}

	// fmt.Printf("Attempting to verify alexandria-media JSON...")
	err := json.Unmarshal(b, &v)
	if err != nil {
		return v, m, err
	}

	errr := json.Unmarshal(b, &i)
	if errr != nil {
		return v, m, err
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
				return v, m, errors.New("can't verify media - JSON object root key doesn't match accepted value")
			}
		}
	}

	fmt.Printf("*** debug: JSON object root matches, printing v:\n%v\n*** /debug ***\n", v)
	// verify torrent hash length
	if len(v.AlexandriaMedia.Torrent) <= 1 {
		return v, m, errors.New("can't verify media - invalid torrent hash length")
	}

	// verify signature
	if v.Signature != signature {
		return v, m, errors.New("can't verify media - signature mismatch")
	}

	// verify timestamp length
	if v.AlexandriaMedia.Timestamp <= 0 {
		return v, m, errors.New("can't verify media - invalid timestamp")
	}

	// verify type length
	if len(v.AlexandriaMedia.Type) <= 1 {
		return v, m, errors.New("can't verify media - invalid type length")
	}

	// verify media info lengths
	if len(v.AlexandriaMedia.Info.Title) <= 0 {
		return v, m, errors.New("can't verify media - invalid info title length")
	}
	if len(v.AlexandriaMedia.Info.Description) <= 0 {
		return v, m, errors.New("can't verify media - invalid info description length")
	}
	if v.AlexandriaMedia.Info.Year <= 0 {
		return v, m, errors.New("can't verify media - invalid info year")
	}
	if v.AlexandriaMedia.Info.Size <= 0 {
		return v, m, errors.New("can't verify media - invalid info size")
	}

	// verify payment info
	if len(v.AlexandriaMedia.Payment.Currency) < 1 {
		return v, m, errors.New("can't verify media - invalid payment currency length")
	}
	if len(v.AlexandriaMedia.Payment.Type) < 1 {
		return v, m, errors.New("can't verify media - invalid payment type length")
	}
	if v.AlexandriaMedia.Payment.Amount <= 0 {
		return v, m, errors.New("can't verify media - invalid payment amount")
	}

	// verify signature was created by this address
	// signature pre-image for media is the string concatenation of torrenthash+timestamp
	if checkSignature(v.AlexandriaMedia.Publisher, signature, v.AlexandriaMedia.Torrent+"-"+v.AlexandriaMedia.Publisher+"-"+strconv.FormatInt(v.AlexandriaMedia.Timestamp, 10)) == false {
		return v, m, errors.New("can't verify media - message failed to pass signature verification")
	}

	// fmt.Println(" -- VERIFIED --")
	return v, m, nil

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

func StoreMedia(media AlexandriaMedia, jmap map[string]interface{}, dbtx *sql.Tx, txid string, block int) {
	// check for media info extras
	extraInfo, ei_err := extractMediaExtraInfo(jmap)
	extraInfoString := ""
	if ei_err != nil {
		fmt.Printf("extra info not found/failed - error returned: %v\n", ei_err)
	} else {
		extraInfoString = string(extraInfo)
	}

	// make sure extras is stored as an empty string if it doesn't exist
	if len(media.AlexandriaMedia.Extras) < 1 {
		media.AlexandriaMedia.Extras = ""
	}

	stmtstr := `insert into media (publisher, torrent, timestamp, type, info_title, info_description, info_year, info_size, info_extra, payment_currency, payment_type, payment_amount, extras, txid, block, signature, multipart, active) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, "` + txid + `", ` + strconv.Itoa(block) + `, ?, 0, 1)`

	stmt, err := dbtx.Prepare(stmtstr)
	if err != nil {
		fmt.Println("exit 102")
		log.Fatal(err)
	}

	fmt.Printf("stmt: %v\n", stmt)

	res, stmterr := stmt.Exec(media.AlexandriaMedia.Publisher, media.AlexandriaMedia.Torrent, media.AlexandriaMedia.Timestamp, media.AlexandriaMedia.Type, media.AlexandriaMedia.Info.Title, media.AlexandriaMedia.Info.Description, media.AlexandriaMedia.Info.Year, media.AlexandriaMedia.Info.Size, extraInfoString, media.AlexandriaMedia.Payment.Currency, media.AlexandriaMedia.Payment.Type, media.AlexandriaMedia.Payment.Amount, media.AlexandriaMedia.Extras, media.Signature)
	if stmterr != nil {
		fmt.Println("exit 103")
		log.Fatal(stmterr)
	}

	fmt.Printf("result: %v\n", res)

	stmt.Close()

}

func checkSignature(address string, signature string, message string) bool {
	if foundation.RPCCall("verifymessage", address, signature, message) == true {
		return true
	}
	return false
}

func extractMediaExtraInfo(jmap map[string]interface{}) ([]byte, error) {
	// find the "extra info" json object
	var ret []byte
	for k, v := range jmap {
		if k == "alexandria-media" {
			vm := v.(map[string]interface{})
			for k2, v2 := range vm {
				if k2 == "info" {
					v2m := v2.(map[string]interface{})
					for k3, v3 := range v2m {
						if k3 == "extra-info" {
							fmt.Printf("v3(%v): %v\n\n", reflect.TypeOf(v3), v3)
							v3json, err := json.Marshal(v3)
							if err != nil {
								return ret, err
							}
							return v3json, nil
						}
					}

				}
			}
		}
	}
	return ret, errors.New("no media extra info found")
}
