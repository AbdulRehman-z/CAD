package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func newEncryptionKey() []byte {
	key := make([]byte, 16)
	_, err := rand.Read(key)
	if err != nil {
		return nil
	}

	return key
}

func copyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}

	var (
		buf    = make([]byte, 32*1024)
		stream = cipher.NewCTR(block, iv)
		nw     = block.BlockSize()
	)

	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nn, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nw += nn
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	return nw, nil
}

func copyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, nil
	}

	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}

	var (
		buf    = make([]byte, 32*1024)
		stream = cipher.NewCTR(block, iv)
	)

	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:0])
			if _, err := dst.Write(buf[:n]); err != nil {
				return 0, err
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return 0, err
			}
		}
		if err != nil {
			return 0, nil
		}
	}

	return 0, nil
}