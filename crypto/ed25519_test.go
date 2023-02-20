// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package crypto

import (
	"crypto/ed25519"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	TestPrivateKey = PrivateKey(
		[PrivateKeyLen]byte{
			32, 241, 118, 222, 210, 13, 164, 128, 3, 18,
			109, 215, 176, 215, 168, 171, 194, 181, 4, 11,
			253, 199, 173, 240, 107, 148, 127, 190, 48, 164,
			12, 48, 115, 50, 124, 153, 59, 53, 196, 150, 168,
			143, 151, 235, 222, 128, 136, 161, 9, 40, 139, 85,
			182, 153, 68, 135, 62, 166, 45, 235, 251, 246, 69, 7,
		},
	)
	TestPublicKey = []byte{
		115, 50, 124, 153, 59, 53, 196, 150, 168, 143, 151, 235,
		222, 128, 136, 161, 9, 40, 139, 85, 182, 153, 68, 135,
		62, 166, 45, 235, 251, 246, 69, 7,
	}
	TestHRP           = "test"
	TestAddressString = "test1wve8exfmxhzfd2y0jl4aaqyg5yyj3z6" +
		"4k6v5fpe75ck7h7lkg5rsf7cgc6" // this is the address associated with TestPublicKey and TestHRP
	TestPrivateKeyHex = "20f176ded20da48003126dd7b0d7a8abc" +
		"2b5040bfdc7adf06b947fbe30a40c3073327c993b35c496a88f97eb" +
		"de8088a109288b55b69944873ea62debfbf64507"
)

func TestGeneratePrivateKeyFormat(t *testing.T) {
	require := require.New(t)
	priv, err := GeneratePrivateKey()
	require.NoError(err, "Error Generating PrivateKey")
	require.NotEqual(priv, EmptyPrivateKey, "PrivateKey is empty")
	require.Len(priv, PrivateKeyLen, "PrivateKey has incorrect length")
}

func TestGeneratePrivateKeyDifferent(t *testing.T) {
	require := require.New(t)
	const numKeysToGenerate int = 10
	pks := [numKeysToGenerate]PrivateKey{}

	// generate keys
	for i := 0; i < numKeysToGenerate; i++ {
		priv, err := GeneratePrivateKey()
		pks[i] = priv
		require.NoError(err, "Error Generating Private Key")
	}

	// make sure keys are different
	m := make(map[PrivateKey]bool)
	for _, priv := range pks {
		require.False(m[priv], "Duplicate PrivateKey generated")
		m[priv] = true
	}
}

func TestPublicKeyValid(t *testing.T) {
	require := require.New(t)
	// Hardcoded test values
	var expectedPubKey PublicKey
	copy(expectedPubKey[:], TestPublicKey)
	pubKey := TestPrivateKey.PublicKey()
	require.Equal(expectedPubKey, pubKey, "PublicKey not equal to Expected PublicKey")
}

func TestPublicKeyFormat(t *testing.T) {
	require := require.New(t)
	priv, err := GeneratePrivateKey()
	require.NoError(err, "Error during call to GeneratePrivateKey")
	pubKey := priv.PublicKey()
	require.NotEqual(pubKey, EmptyPublicKey, "PublicKey is empty")
	require.Len(pubKey, PublicKeyLen, "PublicKey has incorrect length")
}

func TestAddress(t *testing.T) {
	require := require.New(t)
	var pubKey PublicKey
	copy(pubKey[:], TestPublicKey)
	addr := Address(TestHRP, pubKey)
	require.Equal(addr, TestAddressString, "Unexpected Address")
}

func TestParseAddressIncorrectHrt(t *testing.T) {
	require := require.New(t)
	pubkey, err := ParseAddress("wronghrt", TestAddressString)
	require.ErrorIs(err, ErrIncorrectHrp, "ErrIncorrectHrp not returned")
	require.Equal(
		pubkey,
		PublicKey(EmptyPublicKey),
		"Unexpected PublicKey from ParseAddress",
	)
}

func TestParseAddressIncorrectSaddr(t *testing.T) {
	require := require.New(t)

	pubkey, err := ParseAddress(TestHRP, "incorrecttestaddressstring")
	require.Error(err, "Error was not thrown after call with incorrect parameters")
	require.Equal(
		pubkey,
		PublicKey(EmptyPublicKey),
		"Unexpected PublicKey from ParseAddress",
	)
}

func TestParseAddress(t *testing.T) {
	require := require.New(t)

	var expectedPubkey PublicKey
	copy(expectedPubkey[:], TestPublicKey)
	pubkey, err := ParseAddress(TestHRP, TestAddressString)
	require.NoError(err, "Error returned by ParseAddress")
	require.Equal(pubkey, expectedPubkey, "Unexpected PublicKey from ParseAddress")
}

func TestSaveKey(t *testing.T) {
	require := require.New(t)

	filename := "SaveKey"
	err := TestPrivateKey.Save(filename)
	require.NoError(err, "Error during call to SaveKey")
	require.FileExists(filename, "SaveKey did not create file")
	// Check correct key was saved in file
	bytes, err := os.ReadFile(filename)
	var privKey PrivateKey
	copy(privKey[:], bytes)
	require.NoError(err, "Reading saved file threw an error")
	require.Equal(TestPrivateKey, privKey, "Key is different than saved key")
	// Remove File
	os.Remove(filename)
}

func TestLoadKeyIncorrectKey(t *testing.T) {
	// Creates dummy file with invalid key size
	// Checks that LoadKey returns emptyprivatekey and err
	require := require.New(t)
	filename := "TestLoadKey"
	invalidPrivKey := []byte{1, 2, 3, 4, 5}
	// Writes
	err := os.WriteFile(filename, invalidPrivKey, 0o600)
	require.NoError(err, "Error writing using OS during tests")
	privKey, err := LoadKey(filename)
	// Validate
	require.ErrorIs(err, ErrInvalidPrivateKey,
		"ErrInvalidPrivateKey was not returned")
	require.Equal(privKey, PrivateKey(EmptyPrivateKey))
	// Remove file
	os.Remove(filename)
}

func TestLoadKeyInvalidFile(t *testing.T) {
	require := require.New(t)

	filename := "FileNameDoesntExist"
	privKey, err := LoadKey(filename)
	require.Error(err, "Error was not returned")
	require.Equal(privKey, PrivateKey(EmptyPrivateKey),
		"EmptyPrivateKey was not returned")
}

func TestLoadKey(t *testing.T) {
	require := require.New(t)
	// Creates dummy file with valid key size
	// Checks the returned value was the key in the file
	filename := "TestLoadKey"
	// Writes
	err := os.WriteFile(filename, TestPrivateKey[:], 0o600)
	require.NoError(err, "Error writing using OS during tests")

	privKey, err := LoadKey(filename)
	// Validate
	require.NoError(err, "Error was incorrectly returned during LoadKey")
	require.Equal(privKey, TestPrivateKey, "PrivateKey was different than expected")
	// Removes file
	os.Remove(filename)
}

func TestSignSignatureValid(t *testing.T) {
	require := require.New(t)

	msg := []byte("msg")
	// Sign using ed25519
	ed25519Sign := ed25519.Sign(TestPrivateKey[:], msg)
	var expectedSig Signature
	copy(expectedSig[:], ed25519Sign)
	// Sign using crypto
	sig := Sign(msg, TestPrivateKey)
	require.Equal(sig, expectedSig, "Signature was incorrect")
}

func TestVerifyValidParams(t *testing.T) {
	require := require.New(t)
	msg := []byte("msg")
	sig := Sign(msg, TestPrivateKey)
	require.True(Verify(msg, TestPrivateKey.PublicKey(), sig),
		"Signature was invalid")
}

func TestVerifyInvalidParams(t *testing.T) {
	require := require.New(t)

	msg := []byte("msg")

	difMsg := []byte("diff msg")
	sig := Sign(msg, TestPrivateKey)

	require.False(Verify(difMsg, TestPrivateKey.PublicKey(), sig),
		"Verify incorrectly verified a message")
}

func TestHexToKeyInvalidKey(t *testing.T) {
	require := require.New(t)
	invalidHex := "1234"
	hex, err := HexToKey(invalidHex)
	require.ErrorIs(err, ErrInvalidPrivateKey, "Incorrect error returned")
	require.Equal(PrivateKey(EmptyPrivateKey), hex)
}

func TestHexToKey(t *testing.T) {
	require := require.New(t)
	hex, err := HexToKey(TestPrivateKeyHex)
	require.NoError(err, "Incorrect error returned")
	require.Equal(TestPrivateKey, hex)
}

func TestKeyToHex(t *testing.T) {
	require := require.New(t)
	require.Equal(TestPrivateKeyHex, TestPrivateKey.ToHex())
}