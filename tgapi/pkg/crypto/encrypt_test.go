package crypto

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"filippo.io/age"
)

func TestEncrypt(t *testing.T) {
	passphrase := "this is super secret"

	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		t.Errorf("age.NewScryptRecipient() error = %v", err)
		return
	}

	tests := []struct {
		name      string
		plainText string
		wantErr   bool
	}{
		{
			name:      "basic",
			plainText: "testing a secret message",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			encrypter, err := Encrypt(passphrase, buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			_, err = io.Copy(encrypter, strings.NewReader(tt.plainText))
			encrypter.Close()
			if (err != nil) != tt.wantErr {
				t.Errorf("encript plain text error = %v", err)
				return
			}

			r, err := age.Decrypt(bytes.NewReader(buf.Bytes()), identity)
			if err != nil {
				t.Errorf("age.Decrypt() error = %v", err)
				return
			}
			dec, err := ioutil.ReadAll(r)
			if err != nil {
				t.Errorf("ioutil.ReadAll() error = %v", err)
				return
			}
			if string(dec) != tt.plainText {
				t.Errorf("decrypted %q does not equal original", string(dec))
				return
			}
		})
	}
}
