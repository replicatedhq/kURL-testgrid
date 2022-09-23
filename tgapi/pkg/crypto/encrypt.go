package crypto

import (
	"io"

	"filippo.io/age"
)

func Encrypt(passphrase string, dst io.Writer) (io.WriteCloser, error) {
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return nil, err
	}

	encrypter, err := age.Encrypt(dst, recipient)
	if err != nil {
		return nil, err
	}

	return encrypter, nil
}
