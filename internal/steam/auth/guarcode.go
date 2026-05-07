package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

const steamAlphabet = "23456789BCDFGHJKMNPQRTVWXY" // 26 chars

func GenerateSteamGuardCode(sharedSecretB64 string, unixTime int64) (string, error) {
	secret, err := base64.StdEncoding.DecodeString(sharedSecretB64)
	if err != nil {
		return "", fmt.Errorf("decode shared_secret base64: %w", err)
	}

	timeStep := uint64(unixTime / 30)

	// 8-byte big-endian buffer of timestep
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], timeStep)

	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(buf[:])
	sum := mac.Sum(nil) // 20 bytes

	// Dynamic truncation (RFC TOTP-like)
	offset := sum[len(sum)-1] & 0x0F
	codeInt := (int(sum[offset])&0x7F)<<24 |
		(int(sum[offset+1])&0xFF)<<16 |
		(int(sum[offset+2])&0xFF)<<8 |
		(int(sum[offset+3]) & 0xFF)

	// Map to Steam alphabet (5 chars)
	out := make([]byte, 5)
	for i := 0; i < 5; i++ {
		out[i] = steamAlphabet[codeInt%len(steamAlphabet)]
		codeInt /= len(steamAlphabet)
	}
	return string(out), nil
}
