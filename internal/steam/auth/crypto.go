package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
)

// BuildRSAPublicKey строит rsa.PublicKey из hex-модуля и hex-экспоненты.
// expHex часто "010001" => 65537.
func BuildRSAPublicKey(modHex, expHex string) (*rsa.PublicKey, error) {
	nBytes, err := hex.DecodeString(modHex)
	if err != nil {
		return nil, fmt.Errorf("bad modHex: %w", err)
	}
	eBytes, err := hex.DecodeString(expHex)
	if err != nil {
		return nil, fmt.Errorf("bad expHex: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)

	eBig := new(big.Int).SetBytes(eBytes)
	if !eBig.IsInt64() {
		return nil, fmt.Errorf("exponent too large")
	}
	e64 := eBig.Int64()
	if e64 <= 1 || e64 > 1<<31-1 {
		return nil, fmt.Errorf("invalid exponent: %d", e64)
	}

	return &rsa.PublicKey{N: n, E: int(e64)}, nil
}

// EncryptPKCS1v15Base64 шифрует plaintext RSAES-PKCS1-v1_5 и возвращает base64 ciphertext.
// rawB64=true -> без '=' (RawStdEncoding), rawB64=false -> с '=' (StdEncoding).
func EncryptPKCS1v15Base64(pub *rsa.PublicKey, plaintext string, rawB64 bool) (string, error) {
	ct, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(plaintext))
	if err != nil {
		return "", err
	}
	if rawB64 {
		return base64.RawStdEncoding.EncodeToString(ct), nil
	}
	return base64.StdEncoding.EncodeToString(ct), nil
}
