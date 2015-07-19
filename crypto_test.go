package gowork

import (
	"bytes"
	"testing"
)

const (
	ENCRYPT_KEY_VALID   string = "GoWorkGoWorkGoWorkGoWorkGoWork12"
	ENCRYPT_KEY_INVALID string = "GoWork"
	ENCRYPT_TEXT        string = "ExampleText"
)

func TestEncrypt(t *testing.T) {
	var (
		err               error
		buffKey, buffText bytes.Buffer
	)

	if _, err = buffKey.WriteString(ENCRYPT_KEY_VALID); err != nil {
		t.Fatal(err)
	}

	if _, err = buffText.WriteString(ENCRYPT_TEXT); err != nil {
		t.Fatal(err)
	}

	if _, err = encrypt(buffKey.Bytes(), buffText.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func TestEncryptInvalidKeySize(t *testing.T) {
	var (
		err               error
		buffKey, buffText bytes.Buffer
	)

	if _, err = buffKey.WriteString(ENCRYPT_KEY_INVALID); err != nil {
		t.Fatal(err)
	}

	if _, err = buffText.WriteString(ENCRYPT_TEXT); err != nil {
		t.Fatal(err)
	}

	if _, err = encrypt(buffKey.Bytes(), buffText.Bytes()); err == nil {
		t.Fatalf("Expected encrypt to throw a key length error 'invalid key size 6'.  Key %s passed in.", ENCRYPT_KEY_INVALID)
	}
}
