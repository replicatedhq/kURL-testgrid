package crypto

import (
	"bufio"
	"io"

	"filippo.io/age"
)

func StreamEncrypt(password string, plaintext io.Reader) (io.ReadCloser, error) {
	recipient, err := age.NewScryptRecipient(password)
	if err != nil {
		return nil, err
	}
	recipient.SetWorkFactor(15)

	pr, pw := io.Pipe()
	bw := bufio.NewWriter(pw) // buffer age header and nonce
	encrypter, err := age.Encrypt(bw, recipient)
	if err != nil {
		return nil, err
	}

	go func() {
		_, err := io.Copy(encrypter, plaintext)
		encrypter.Close()
		bw.Flush()
		pw.CloseWithError(err)
	}()

	return pr, nil
}
