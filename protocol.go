package alexandriaMedia

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/metacoin/flojson"
	"github.com/metacoin/foundation"
)

var ()

const MEDIA_ROOT_KEY = "alexandria-media"
const PUBLISHER_ROOT_KEY = "alexandria-publisher"
const MIN_BLOCK = 1002555

// media structs
type AlexandriaMedia struct {
	AlexandriaMedia struct {
		// required media metadata
		Torrent   string `json:"torrent"`
		Publisher string `json:"publisher"`
		Timestamp int64  `json:"timestamp"`
		Type      string `json:"type"`

		Info struct {
			// required file information
			Title       string `json:"title"`
			Description string `json:"description"`
			Year        int    `json:"year"`

			// optional extra-info field
			ExtraInfo interface{} `json:"extra-info"`
		} `json:"info"`

		// optional fields
		Payment interface{} `json:"payment"`
		Extras  string      `json:"extras"`
	} `json:"alexandria-media"`
	Signature string `json:"signature"`
}

type AlexandriaPublisher struct {
	AlexandriaPublisher struct {
		// required publisher metadata
		Name      string `json:"name"`
		Address   string `json:"address"`
		Timestamp int64  `json:"timestamp"`

		// optional fields
		Emailmd5   string `json:"emailmd5"`
		Bitmessage string `json:"bitmessage"`
	} `json:"alexandria-publisher"`
	Signature string `json:"signature"`
}

// multipart struct
type MediaMultipartSingle struct {
	Part      int
	Max       int
	Reference string
	Address   string
	Signature string
	Data      string
	Txid      string
	Block     int
}

// reference: Cory LaNou, Mar 2 '14 at 15:21, http://stackoverflow.com/a/22129435/2576956
func IsJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func VerifyPublisher(b []byte) (AlexandriaPublisher, error) {

	var v AlexandriaPublisher
	var i interface{}
	var m map[string]interface{}

	// fmt.Printf("Attempting to verify alexandria-publisher JSON...")

	if !IsJSON(string(b)) {
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
	// signature pre-image for publisher is <name>-<address>-<timestamp>
	if checkSignature(v.AlexandriaPublisher.Address, signature, v.AlexandriaPublisher.Name+"-"+v.AlexandriaPublisher.Address+"-"+strconv.FormatInt(v.AlexandriaPublisher.Timestamp, 10)) == false {
		return v, errors.New("can't verify publisher - message failed to pass signature verification")
	}

	// fmt.Println(" -- VERIFIED --")
	return v, nil

}

func StoreMediaMultipartSingle(mms MediaMultipartSingle, dbtx *sql.Tx) {
	// store in database
	stmtstr := `insert into media_multipart (part, max, address, reference, signature, data, txid, block, complete, success, active) values (` + strconv.Itoa(mms.Part) + `, ` + strconv.Itoa(mms.Max) + `, ?, ?, ?, ?, "` + mms.Txid + `", ` + strconv.Itoa(mms.Block) + `, 0, 0, 1)`

	stmt, err := dbtx.Prepare(stmtstr)
	if err != nil {
		fmt.Println("exit 160")
		log.Fatal(err)
	}

	_, stmterr := stmt.Exec(mms.Address, mms.Reference, mms.Signature, mms.Data)
	if stmterr != nil {
		fmt.Println("exit 161")
		log.Fatal(stmterr)
	}

	stmt.Close()

}

func CheckPublisherAddressExists(address string, dbtx *sql.Tx) bool {
	// check if this publisher address is already in-use
	stmtstr := `select name from publisher where address = ?`

	rows, stmterr := dbtx.Query(stmtstr, address)
	if stmterr != nil {
		fmt.Println("exit 91248")
		log.Fatal(stmterr)
	}

	var rowsCount int = 0
	for rows.Next() {
		rowsCount++
	}

	rows.Close()
	return rowsCount > 0

}

func CheckMediaMultipartComplete(reference string, dbtx *sql.Tx) ([]byte, error) {
	// using the reference tx, check how many different txs we have and determine if we have all transactions
	// if we have a valid media-multipart complete instance, let's return the byte array it consists of
	var ret []byte

	stmtstr := `select part, max, data from media_multipart where active = 1 and complete = 0 and reference = "` + reference + `" order by part asc`

	stmt, err := dbtx.Prepare(stmtstr)
	if err != nil {
		fmt.Println("exit 120")
		log.Fatal(err)
	}

	rows, stmterr := stmt.Query()
	if err != nil {
		fmt.Println("exit 121")
		log.Fatal(stmterr)
	}

	var rowsCount int = 0
	var pmax int
	var fullData string

	for rows.Next() {
		var part int
		var max int
		var data string
		rows.Scan(&part, &max, &data)

		// TODO: require signature verification for multipart messages
		if rowsCount > max {
			return ret, errors.New("too many rows in multipart message - check for reorg/bogus multipart data")
		}
		rowsCount++

		pmax = max
		fullData += data
	}

	if rowsCount != pmax+1 {
		return ret, errors.New("only found " + strconv.Itoa(rowsCount) + "/" + strconv.Itoa(pmax+1) + " multipart messages")
	}

	stmt.Close()
	rows.Close()

	// set complete to 1
	updatestr := `update media_multipart set complete = 1 where reference = "` + reference + `"`
	updatestmt, updateerr := dbtx.Prepare(updatestr)
	if updateerr != nil {
		fmt.Println("exit 122")
		log.Fatal(updateerr)
	}

	_, updatestmterr := updatestmt.Exec()
	if updatestmterr != nil {
		fmt.Println("exit 123")
		log.Fatal(updatestmterr)
	}
	updatestmt.Close()

	return []byte(fullData), nil
}

func UpdateMediaMultipartSuccess(reference string, dbtx *sql.Tx) {

	stmtstr := `update media_multipart set success = 1 where reference = "` + reference + `"`

	stmt, err := dbtx.Prepare(stmtstr)
	if err != nil {
		fmt.Println("exit 140")
		log.Fatal(err)
	}

	_, stmterr := stmt.Exec()
	if err != nil {
		fmt.Println("exit 141")
		log.Fatal(stmterr)
	}

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

	// check length
	if len(s) < 108 {
		return ret, errors.New("not enough data in mutlipart string")
	}

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

	// get and check address
	address := s[4:38]
	if !CheckAddress(address) {
		// fmt.Println("address doesn't check out: \"" + address + "\"")
		return ret, errors.New("address doesn't validate using validateaddress")
	}

	// get reference txid
	reference := s[39:103]

	// get and check signature
	sigEndIndex := strings.Index(s, "):")

	if sigEndIndex == -1 {
		fmt.Println("no end of signature found, malformed tx-comment")
		return ret, errors.New("no end of signature found, malformed tx-comment")
	}

	signature := s[104:sigEndIndex]
	data := s[sigEndIndex+2:]
	// fmt.Println("data: \"" + data + "\"")

	// signature pre-image is <part>-<max>-<address>-<txid>-<data>
	// in the case of multipart[0], txid is 64 zeros
	// in the case of multipart[n], where n != 0, txid is the reference txid (from multipart[0])
	preimage := string(s[0]) + "-" + string(s[2]) + "-" + address + "-" + reference + "-" + data
	// fmt.Printf("preimage: %v\n", preimage)

	if !checkSignature(address, signature, preimage) {
		// fmt.Println("signature didn't pass checksignature test")
		return ret, errors.New("signature didn't pass checksignature test")
	}

	// if part == 0, reference should be submitted in the tx-comment as a string of 64 zeros
	// the local DB will store reference = txid for this transaction after it's submitted
	// in case of a reorg, the publisher must re-publish this multipart message (sorry)
	if part == 0 {
		if reference != "0000000000000000000000000000000000000000000000000000000000000000" {
			// fmt.Println("reference txid should be 64 zeros for part 0 of a multipart message")
			return ret, errors.New("reference txid should be 64 zeros for part 0")
		}
		reference = txid
	}
	// all checks passed, verified!

	//fmt.Printf("data: %v\n", data)
	// fmt.Printf("=== VERIFIED ===\n")
	//fmt.Printf("part: %v\nmax: %v\nreference: %v\naddress: %v\nsignature: %v\ntxid: %v\nblock: %v\n", part, max, reference, address, signature, txid, block)

	ret = MediaMultipartSingle{
		Part:      part,
		Max:       max,
		Reference: reference,
		Address:   address,
		Signature: signature,
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

	// fmt.Printf("*** debug: JSON object root matches, printing v:\n%v\n*** /debug ***\n", v)
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

	// verify signature was created by this address
	// signature pre-image for media is <torrenthash>-<publisher>-<timestamp>
	if checkSignature(v.AlexandriaMedia.Publisher, signature, v.AlexandriaMedia.Torrent+"-"+v.AlexandriaMedia.Publisher+"-"+strconv.FormatInt(v.AlexandriaMedia.Timestamp, 10)) == false {
		return v, m, errors.New("can't verify media - message failed to pass signature verification")
	}

	// fmt.Println(" -- VERIFIED --")
	return v, m, nil

}

func StorePublisher(publisher AlexandriaPublisher, dbtx *sql.Tx, txid string, block int, hash string) {
	// store in database
	stmtstr := `insert into publisher (name, address, timestamp, txid, block, emailmd5, bitmessage, hash, signature, active) values (?, ?, ?, "` + txid + `", ` + strconv.Itoa(block) + `, ?, ?, "` + hash + `", ?, 1)`

	stmt, err := dbtx.Prepare(stmtstr)
	if err != nil {
		fmt.Println("exit 100")
		log.Fatal(err)
	}

	_, stmterr := stmt.Exec(publisher.AlexandriaPublisher.Name, publisher.AlexandriaPublisher.Address, publisher.AlexandriaPublisher.Timestamp, publisher.AlexandriaPublisher.Emailmd5, publisher.AlexandriaPublisher.Bitmessage, publisher.Signature)
	if err != nil {
		fmt.Println("exit 101")
		log.Fatal(stmterr)
	}

	stmt.Close()

}

func StoreMedia(media AlexandriaMedia, jmap map[string]interface{}, dbtx *sql.Tx, txid string, block int, multipart int) {
	// check for media payment data
	payment, payment_err := extractMediaPayment(jmap)
	paymentString := ""
	if payment_err != nil {
		fmt.Printf("payment data not found/failed - error returned: %v\n", payment_err)
	} else {
		paymentString = string(payment)
	}

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

	stmtstr := `insert into media (publisher, torrent, timestamp, type, info_title, info_description, info_year, info_extra, payment, extras, txid, block, signature, multipart, active) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, "` + txid + `", ` + strconv.Itoa(block) + `, ?, ` + strconv.Itoa(multipart) + `, 1)`

	stmt, err := dbtx.Prepare(stmtstr)
	if err != nil {
		fmt.Println("exit 102")
		log.Fatal(err)
	}

	// fmt.Printf("stmt: %v\n", stmt)

	_, stmterr := stmt.Exec(media.AlexandriaMedia.Publisher, media.AlexandriaMedia.Torrent, media.AlexandriaMedia.Timestamp, media.AlexandriaMedia.Type, media.AlexandriaMedia.Info.Title, media.AlexandriaMedia.Info.Description, media.AlexandriaMedia.Info.Year, extraInfoString, paymentString, media.AlexandriaMedia.Extras, media.Signature)
	if stmterr != nil {
		fmt.Println("exit 103")
		log.Fatal(stmterr)
	}

	stmt.Close()

}

func CreateNewPublisherTxComment(b []byte) {
	// given some JSON, post it to the blockchain using either a tx-comment or multipart tx-comment

}

// some helper functions here
func checkSignature(address string, signature string, message string) bool {
	reply, err := foundation.RPCCall("verifymessage", address, signature, message)
	if err != nil {
		fmt.Println("foundation error: " + err.Error())
		return false
	}
	if reply == true {
		return true
	}
	return false
}

func CheckAddress(address string) bool {
	reply, err := foundation.RPCCall("validateaddress", address)
	if err != nil {
		fmt.Println("foundation error: " + err.Error())
		return false
	}
	result, ok := reply.(*flojson.ValidateAddressResult)
	if !ok {
		return false
	}
	if result.IsValid == true {
		return true
	}
	return false
}

func extractMediaPayment(jmap map[string]interface{}) ([]byte, error) {
	// find the "payment" json object
	var ret []byte
	for k, v := range jmap {
		if k == "alexandria-media" {
			vm := v.(map[string]interface{})
			for k2, v2 := range vm {
				if k2 == "payment" {
					// fmt.Printf("v3(%v): %v\n\n", reflect.TypeOf(v3), v3)
					v2json, err := json.Marshal(v2)
					if err != nil {
						return ret, err
					}
					return v2json, nil

				}
			}
		}
	}
	return ret, errors.New("no payment extra info found")
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
							// fmt.Printf("v3(%v): %v\n\n", reflect.TypeOf(v3), v3)
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
