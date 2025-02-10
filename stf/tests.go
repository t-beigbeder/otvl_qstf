package stf

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/t-beigbeder/otvl_qstf/internal/bfio"
)

func toJsonBytes(a any) ([]byte, error) {
	bs, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	wbs := make([]byte, len(bs)+4)
	binary.BigEndian.PutUint32(wbs, uint32(len(bs)))
	copy(wbs[4:], bs)
	return wbs, err
}

func fromJsonBytes(wr bfio.BufWriter, a any) error {
	obs := wr.Bytes()
	bln := binary.BigEndian.Uint32(obs[0:4])
	if bln != uint32(len(obs)-4) {
		return fmt.Errorf("invalid number of bytes read: expected %d got %d", len(obs)-4, bln)
	}
	err := json.Unmarshal(obs[4:], &a)
	if err != nil {
		return err
	}
	return nil
}
