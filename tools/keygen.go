package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"fmt"
)

// To encode publicKey use:
// publicKeyBytes, _ = x509.MarshalPKIXPublicKey(&private_key.PublicKey)

func main() {
	p521 := elliptic.P521()
	priv1, _ := ecdsa.GenerateKey(p521, rand.Reader)

	privateKeyBytes, _ := x509.MarshalECPrivateKey(priv1)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&priv1.PublicKey)

	encodedPriv := hex.EncodeToString(privateKeyBytes)
	encodedPub := hex.EncodeToString(publicKeyBytes)
	fmt.Printf("Encoded private key is:\n%s\n", encodedPriv)
	fmt.Printf("Encoded public key is:\n%s\n", encodedPub)
}
