package crypto

import (
	"io"
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
			name:      "short",
			plainText: "testing a secret message",
		},
		{
			name:      "long",
			plainText: strings.Repeat("testing a secret message", 10000),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := StreamEncrypt(passphrase, strings.NewReader(tt.plainText))
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			r, err := age.Decrypt(enc, identity)
			if err != nil {
				t.Errorf("age.Decrypt() error = %v", err)
				return
			}
			dec, err := io.ReadAll(r)
			enc.Close()
			if err != nil {
				t.Errorf("io.ReadAll() error = %v", err)
				return
			}
			if string(dec) != tt.plainText {
				t.Errorf("decrypted %q does not equal original", string(dec))
				return
			}
		})
	}
}
