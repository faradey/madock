package helper

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"hash"
	"hash/fnv"
	"io"
	"os"
	"strings"
)

func Hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func HashFile(path string, hashType string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = f.Close()
	}()

	buf := make([]byte, 1024*1024)
	h, err := selectHash(hashType)
	if err != nil {
		return "", err
	}

	for {
		bytesRead, err := f.Read(buf)
		if err != nil {
			if err != io.EOF {
				return "", err
			}
			_, err = h.Write(buf[:bytesRead])
			if err != nil {
				return "", err
			}
			break
		}
		_, err = h.Write(buf[:bytesRead])
		if err != nil {
			return "", err
		}
	}

	fileHash := hex.EncodeToString(h.Sum(nil))
	return fileHash, nil
}

func selectHash(hashType string) (hash.Hash, error) {
	switch strings.ToLower(hashType) {
	case "md5":
		return md5.New(), nil
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	}
	return nil, errors.New("Unknown hash: " + hashType)
}
