package common

import (
	"bytes"
	"encoding/json"
	"filippo.io/age"
	"fmt"
	"io"
	"strings"
)

func NewKeyPair() (string, string, error) {
	xi, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", err
	}
	return xi.Recipient().String(), xi.String(), nil
}

func EncryptMsg(msg string, srs ...string) ([]byte, error) {
	bsa := bytes.Buffer{}
	var rs []age.Recipient
	for _, sr := range srs {
		r, err := age.ParseX25519Recipient(sr)
		if err != nil {
			return nil, fmt.Errorf("in EncryptMsg: %w", err)
		}
		rs = append(rs, r)
	}
	wc, err := age.Encrypt(&bsa, rs...)
	if err != nil {
		return nil, fmt.Errorf("in EncryptMsg: %w", err)
	}
	_, err = io.Copy(wc, strings.NewReader(msg))
	if err != nil {
		return nil, fmt.Errorf("in EncryptMsg: %w", err)
	}
	err = wc.Close()
	if err != nil {
		return nil, fmt.Errorf("in EncryptMsg: %w", err)
	}
	bsb, err := json.Marshal(bsa.Bytes())
	if err != nil {
		return nil, fmt.Errorf("in EncryptMsg: %w", err)
	}
	return bsb, nil
}

func DecryptMsg(jbs []byte, sids ...string) (string, error) {
	var bs []byte
	err := json.Unmarshal(jbs, &bs)
	if err != nil {
		return "", fmt.Errorf("in DecryptMsg: %w", err)
	}
	var ids []age.Identity
	for _, sid := range sids {
		id, err := age.ParseX25519Identity(sid)
		if err != nil {
			return "", fmt.Errorf("in DecryptMsg: %w", err)
		}
		ids = append(ids, id)
	}
	rd, err := age.Decrypt(bytes.NewReader(bs), ids...)
	if err != nil {
		return "", fmt.Errorf("in DecryptMsg: %w", err)
	}
	bss, err := io.ReadAll(rd)
	if err != nil {
		return "", fmt.Errorf("in DecryptMsg: %w", err)
	}
	return string(bss), nil
}

func Encrypt(dst io.Writer, srs ...string) (io.WriteCloser, error) {
	var rs []age.Recipient
	for _, sr := range srs {
		r, err := age.ParseX25519Recipient(sr)
		if err != nil {
			return nil, fmt.Errorf("in EncryptMsg: %w", err)
		}
		rs = append(rs, r)
	}
	return age.Encrypt(dst, rs...)
}
func Decrypt(src io.Reader, sids ...string) (io.Reader, error) {
	var ids []age.Identity
	for _, sid := range sids {
		id, err := age.ParseX25519Identity(sid)
		if err != nil {
			return nil, fmt.Errorf("in DecryptMsg: %w", err)
		}
		ids = append(ids, id)
	}
	return age.Decrypt(src, ids...)
}
