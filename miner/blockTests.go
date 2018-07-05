package miner

/*
import (
	"../shared"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

type TestParam struct{
	PrivateKey *ecdsa.PrivateKey
	PubKeyStr string
	Op Op
	Block
}

var testArgs TestParam

func HashOpTest() {

	hashedBytes := testArgs.Op.HashToBytes()
	if len(hashedBytes) != 16 {
		exitOnError("HashOp", errors.New("incorrect length of hashed bytes"))
	}
}

func FindNonceTest() {
	block := FindNonce(testArgs.Block)
	if block != nil {
		fmt.Println(block)
	}
}

func exitOnError(prefix string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s, err = %s\n", prefix, err.Error())
		os.Exit(1)
	}
}

// init var
func init(){

	// Generate pub/priv keys
	p256 := elliptic.P256()
	testArgs.PrivateKey, _ = ecdsa.GenerateKey(p256, rand.Reader)
	pubKeyBytes, _ := x509.MarshalPKIXPublicKey(&testArgs.PrivateKey.PublicKey)
	testArgs.PubKeyStr = hex.EncodeToString(pubKeyBytes)

	privKeyStr, err := shared.EncodePrivKey(testArgs.PrivateKey)
	fmt.Println(testArgs.PubKeyStr)
	fmt.Println(privKeyStr)

	// Init Op
	shape := Shape{svg: "M 0 0 H 20 V 20 h -20 Z"}
	// OpSig: Sign the message with private key
	r, s, err := ecdsa.Sign(rand.Reader, testArgs.PrivateKey, shape.HashToBytes())
	if err != nil {
		exitOnError("error during signing", err)
	}
	opSig := shared.OpenArgs{r, s}
	testArgs.Op = Op{
		Op:       shape,
		OpSig:    opSig,
		PubKey:   testArgs.PubKeyStr,
		ValidNum: 4,
		Add: true,
	}
	ops := make([]Op, 1)
	ops[0] = testArgs.Op

	// Init block
	testArgs.Block = OpBlock{
		PrevHash: "83218ac34c1834c26781fe4bde918ee4", //Genesis block
		Ops:         ops,
		PubKeyMiner: testArgs.PubKeyStr,
		Nonce:       0,
	}

	//FindNonceTest()

}
*/
