package shared

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/md5"
	"crypto/x509"
	"encoding/gob"
	"encoding/hex"
	"math/rand"
	"time"
)

// v is the object to be serialized
func Serialize(v interface{}) []byte {
	buff := new(bytes.Buffer)
	encoder := gob.NewEncoder(buff)
	encoder.Encode(v)
	return buff.Bytes()
}

// b - raw data
// v - pointer to object to store decoded data
func Deserialize(b []byte, vp interface{}) {
	buff := bytes.NewBuffer(b)
	decoder := gob.NewDecoder(buff)
	decoder.Decode(vp)
}

// Decode hex encoded private key string (contains public key)
func DecodePrivKey(privKeyStr string) (*ecdsa.PrivateKey, error) {
	privKeyHex, err := hex.DecodeString(privKeyStr)
	if err != nil {
		return nil, err
	}

	return x509.ParseECPrivateKey(privKeyHex)
}

func EncodePrivKey(privKey *ecdsa.PrivateKey) (privKeyStr string, err error) {
	privateKeyBytes, err := x509.MarshalECPrivateKey(privKey)
	privKeyStr = hex.EncodeToString(privateKeyBytes)
	return
}

// Decode hex encoded public key string
// TODO: need to test
func DecodePubKey(pubKeyStr string) (*ecdsa.PublicKey, error) {
	pubKeyHex, err := hex.DecodeString(pubKeyStr)
	if err != nil {
		return nil, err
	}
	re, err := x509.ParsePKIXPublicKey(pubKeyHex)
	pubKey := re.(*ecdsa.PublicKey)
	return pubKey, err
}

func EncodePubKey(pubKey ecdsa.PublicKey) (pubKeyStr string, err error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&pubKey)
	pubKeyStr = hex.EncodeToString(publicKeyBytes)
	return
}

// Converts byte array to hashed byte array using md5
func HashByteArr(data []byte) (hashedData []byte) {
	h := md5.New()
	h.Write(data)
	hashedData = h.Sum(nil)
	return hashedData
}

func ConcateByteArr(arrs [][]byte) (arr []byte) {
	for _, data := range arrs {
		arr = append(arr, data[:]...)
	}
	return arr
}

// Random number generator
func RandomNumGenerator() uint32 {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	randNum := r1.Int31()

	return uint32(randNum)
}
