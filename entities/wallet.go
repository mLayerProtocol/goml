package entities

import (
	// "errors"

	"encoding/json"
	"strings"

	"github.com/mlayerprotocol/go-mlayer/common/encoder"
	"github.com/mlayerprotocol/go-mlayer/internal/crypto"
	"gorm.io/gorm"
)



type Wallet struct {
	// Primary
	ID string `gorm:"primaryKey;type:uuid;not null" json:"id,omitempty"`
	Account           AddressString `json:"acct"`
	Subnet             string        `json:"subn" gorm:"type:varchar(32);index;not null" msgpack:",noinline"`
	Name         string        `json:"name" gorm:"type:char(12);not null"`
	Timestamp            uint64                           `json:"ts"`
}

func (d *Wallet) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == "" {
		uuid, err := GetId(*d)
		if err != nil {
			logger.Error(err)
			panic(err)
		}

		d.ID = uuid
	}
	
	return nil
}

// func (e *Wallet) Key() string {
// 	return fmt.Sprintf("/%s", e.Hash)
// }

func (e *Wallet) ToJSON() []byte {
	m, err := json.Marshal(e)
	if err != nil {
		logger.Errorf("Unable to parse event to []byte")
	}
	return m
}

func (e *Wallet) MsgPack() []byte {
	b, _ := encoder.MsgPackStruct(e)
	return b
}


func WalletFromJSON(b []byte) (Event, error) {
	var e Event
	// if err := json.Unmarshal(b, &message); err != nil {
	// 	panic(err)
	// }
	err := json.Unmarshal(b, &e)
	return e, err
}

func (e Wallet) GetHash() ([]byte, error) {
	b, err := e.EncodeBytes()
	if err != nil {
		return []byte(""), err
	}
	return crypto.Sha256(b), nil
}

func (e Wallet) ToString() string {
	values := []string{}
	values = append(values, e.Name)
	values = append(values, e.Subnet)
	values = append(values,  e.Account.ToString())
	
	return strings.Join(values, "")
}

func (e Wallet) EncodeBytes() ([]byte, error) {

	
	return encoder.EncodeBytes(
		encoder.EncoderParam{Type: encoder.StringEncoderDataType, Value: e.Name},
		encoder.EncoderParam{Type: encoder.HexEncoderDataType, Value: e.Subnet},
		encoder.EncoderParam{Type: encoder.StringEncoderDataType, Value: e.Account},
	)
}

