package gowork

import (
	"bytes"
	"testing"
)

const (
	DECRYPT_TEXT_INVALID string = "GoWork"
	DECRYPT_TEXT_VALID   string = "GoWorkGoWork1234"
	ENCRYPT_KEY_VALID    string = "GoWorkGoWorkGoWorkGoWorkGoWork12"
	ENCRYPT_KEY_INVALID  string = "GoWork"
	ENCRYPT_TEXT         string = "ExampleText"
)

func TestDecrypt(t *testing.T) {
	var (
		err               error
		buffKey, buffText bytes.Buffer
	)

	t.Parallel()

	if _, err = buffKey.WriteString(ENCRYPT_KEY_VALID); err != nil {
		t.Fatal(err)
	}

	if _, err = buffText.WriteString(DECRYPT_TEXT_VALID); err != nil {
		t.Fatal(err)
	}

	if _, err = decrypt(buffKey.Bytes(), buffText.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func TestDecryptInvalidKeySize(t *testing.T) {
	var (
		err               error
		buffKey, buffText bytes.Buffer
	)

	t.Parallel()

	if _, err = buffKey.WriteString(ENCRYPT_KEY_INVALID); err != nil {
		t.Fatal(err)
	}

	if _, err = buffText.WriteString(DECRYPT_TEXT_INVALID); err != nil {
		t.Fatal(err)
	}

	if _, err = decrypt(buffKey.Bytes(), buffText.Bytes()); err == nil {
		t.Fatalf("Expected encrypt to throw a key length error 'invalid key size 6'.  Key %s passed in.", ENCRYPT_KEY_INVALID)
	}
}

func TestEncrypt(t *testing.T) {
	var (
		err               error
		buffKey, buffText bytes.Buffer
	)

	t.Parallel()

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

	t.Parallel()

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
