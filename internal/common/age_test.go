package common

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t-beigbeder/otvl_qstf/internal/bfio"
	"io"
	"testing"
)

func TestAgeEncDec(t *testing.T) {
	pub, pri, err := NewKeyPair()
	require.NoError(t, err)
	ebs, err := EncryptMsg("TestAgeEncDec", pub)
	require.NoError(t, err)
	dbs, err := DecryptMsg(ebs, pri)
	require.NoError(t, err)
	require.Equal(t, "TestAgeEncDec", dbs)
}

func TestAgeEncDecBinaryStream(t *testing.T) {
	pub, pri, err := NewKeyPair()
	require.NoError(t, err)
	wr := bfio.NewBufWr()
	ewr, err := Encrypt(wr, pub)
	require.NoError(t, err)
	in := bytes.NewReader([]byte("TestAgeEncDecBinaryStream"))
	_, err = io.Copy(ewr, in)
	require.NoError(t, err)
	err = ewr.Close()
	require.NoError(t, err)
	_ = pri
	in = bytes.NewReader(wr.Bytes())
	rr, err := Decrypt(in, pri)
	require.NoError(t, err)
	wr2 := bfio.NewBufWr()
	_, err = io.Copy(wr2, rr)
	require.NoError(t, err)
	assert.Equal(t, []byte("TestAgeEncDecBinaryStream"), wr2.Bytes())
}
