package p2p

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/mlayerprotocol/go-mlayer/common/apperror"
	"github.com/mlayerprotocol/go-mlayer/common/encoder"
	"github.com/mlayerprotocol/go-mlayer/common/utils"
	"github.com/mlayerprotocol/go-mlayer/configs"
	"github.com/mlayerprotocol/go-mlayer/entities"
	"github.com/mlayerprotocol/go-mlayer/internal/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)
type P2pAction int8

type P2pEventResponse struct {
	Event json.RawMessage `json:"e"`
	States []json.RawMessage `json:"s"`
}

func (hs *P2pEventResponse) MsgPack() []byte {
	b, _ := encoder.MsgPackStruct(hs)
	return b
}

func (hs P2pEventResponse) Unpack(b []byte) (error) {
	return encoder.MsgPackUnpackStruct(b, &hs)
}

func UnpackP2pEventResponse(b []byte) ( P2pEventResponse, error) {
	var message  P2pEventResponse
	err := encoder.MsgPackUnpackStruct(b, &message)
	return message, err
}


const (
	P2pActionResponse P2pAction = 0
	P2pActionGetEvent P2pAction = 1
	P2pActionGetCommitment P2pAction = 2
	P2pActionGetSentryProof P2pAction = 3
	P2pActionGetTokenProof P2pAction = 4
	P2pActionGetState P2pAction = 5
	P2pActionSyncBlock P2pAction = 6
	
)
type P2pPayload struct {
	// Messages is a channel of messages received from other peers in the chat channel
	Id string `json:"id"`
	Data json.RawMessage `json:"d"`
	ChainId configs.ChainId `json:"pre"`
	Timestamp uint64 `json:"ts"`
	Action P2pAction `json:"ac"`
	ResponseCode apperror.ErrorCode `json:"resp"`
	Error string `json:"err"`
	Signature json.RawMessage  `json:"sig"`
	Signer json.RawMessage  `json:"sign"`
	config *configs.MainConfiguration `json:"-" msgpack:"-"`
}

func (hs *P2pPayload) MsgPack() []byte {
	b, _ := encoder.MsgPackStruct(hs)
	return b
}

func (hsd P2pPayload) EncodeBytes() ([]byte, error) {
    return encoder.EncodeBytes(
		encoder.EncoderParam{Type: encoder.IntEncoderDataType, Value: hsd.Action},
		encoder.EncoderParam{Type: encoder.ByteEncoderDataType, Value: hsd.Data},
		encoder.EncoderParam{Type: encoder.StringEncoderDataType, Value: hsd.Id},
        encoder.EncoderParam{Type: encoder.ByteEncoderDataType, Value: hsd.ChainId.Bytes()},
		encoder.EncoderParam{Type: encoder.IntEncoderDataType, Value: hsd.ResponseCode},
		encoder.EncoderParam{Type: encoder.IntEncoderDataType, Value: hsd.Timestamp},
	)
}
func (nma * P2pPayload) IsValid(chainId configs.ChainId) bool {
	// Important security update. Do not remove. 
	// Prevents cross chain replay attack
	if nma == nil || len(nma.Data) == 0 {
		return false
	}
	nma.ChainId = chainId  // Important security update. Do not remove
	//
	if math.Abs(float64(uint64(time.Now().UnixMilli()) - nma.Timestamp)) > float64(15 * time.Second.Milliseconds()) {
		logger.WithFields(logrus.Fields{"data": nma}).Warnf("P2pPayload: Expired -> %d", uint64(time.Now().UnixMilli()) - nma.Timestamp)
		return false
	}
	// signer, err := hex.DecodeString(string(nma.Signer));
	// if err != nil {
	// 	logger.Error("Unable to decode signer")
	// 	return false
	// }
	
	data, err := nma.EncodeBytes()
	if err != nil {
		logger.Error("Unable to decode signer")
		return false
	}
	
	isValid, err := crypto.VerifySignatureEDD(nma.Signer, &data, nma.Signature)
	if err != nil {
		logger.Error(err)
		return false
	}
	
	if !isValid {
		logger.WithFields(logrus.Fields{"address": nma.Signer, "signature": hex.EncodeToString(nma.Signature)}).Warnf("Invalid signer %s", nma.Signer)
		return false
	}
	return true
}
type RequestType int8
const (
	DataRequest RequestType = 1
	SyncRequest RequestType = 2
)
func (p *P2pPayload) SendDataRequest(receiverPublicKey string) (*P2pPayload, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()
	
	address, err := GetNodeAddress(p.config.Context, receiverPublicKey)
	if err != nil {
		return nil, fmt.Errorf("p2p.GetNodeAddress: %v", err)
	}
	
	return p.SendRequestToAddress(p.config.PrivateKeyEDD, address, DataRequest)
}
func (p *P2pPayload) SendSyncRequest(receiverPublicKey string) (*P2pPayload, error) {
	p.Action = P2pActionSyncBlock
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()
	address, err := GetNodeAddress(p.config.Context, receiverPublicKey)
	if err != nil {
		return nil, fmt.Errorf("p2p.GetNodeAddress: %v", err)
	}
	
	return p.SendRequestToAddress(p.config.PrivateKeyEDD, address, SyncRequest)
}
func (p *P2pPayload) SendRequestToAddress(privateKey []byte, address multiaddr.Multiaddr, _type RequestType) (*P2pPayload, error) {
	p.Sign(privateKey)
	peer,  dataStream, syncStream, err := connectToNode(address, *p.config.Context)
	if err != nil {
		return nil, fmt.Errorf("P2pPayload: %v", err)
	}
	logger.Infof("Preparing to send paylaod to peer: %s", p.Id)
	stream := dataStream
	if _type == SyncRequest {
		stream = syncStream
	}
	if stream != nil {
		rw := bufio.NewReadWriter(bufio.NewReader(*stream), bufio.NewWriter(*stream))
		// handlePayload(*stream)
		s := (*stream)
		// defer s.Close()
		i, err := rw.Write(append(p.MsgPack(), Delimiter...))
		rw.Flush()
		logger.Infof("BytesWritten: %d", i)

		if err != nil {
			if err == network.ErrReset {
				//TODO reconnect
				return nil, err
			}
			logger.Error(err)
			return nil, err
		}
		
		// s.SetReadDeadline(time.Now().Add(60 * time.Second))
		var payloadBuf bytes.Buffer
		bufferLen := 1024
		buf := make([]byte, bufferLen)
		
		for {
			
			n, err := s.Read(buf)
			
			if n > 0 {
				payloadBuf.Write(buf[:n])
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			if n < bufferLen {
				break;
			}
		}
		
		if payloadBuf.Len() == 0 {
			return nil, apperror.Unauthorized("response is invalid")
		}
		
		resp, err := UnpackP2pPayload(payloadBuf.Bytes()[:payloadBuf.Len()-1])
		if err != nil {
			logger.Infof("UnpackREadBYtes: %v", err)
			return resp, err
		}
		logger.Infof("REadBYtes 1: %v", err)
		if !resp.IsValid(p.ChainId) {
			return nil, apperror.Unauthorized("response is invalid")
		}
		return resp, nil	
	}
	logger.Infof("Failed to connect")
	return nil, fmt.Errorf("P2pPayload: connection failed %s", peer.ID)
}

func NewP2pPayload(config *configs.MainConfiguration, action P2pAction, data []byte) (*P2pPayload) {
	pl := P2pPayload{Action: action, Data: data,  Id: utils.RandomString(12), ChainId: config.ChainId}
	pl.config = config
	return &pl
}

func GetState(config *configs.MainConfiguration, path entities.EntityPath,  validator *entities.PublicKeyString, result any) (*P2pEventResponse, error) {
	
	pl := P2pPayload{Action: P2pActionGetState, Data: path.MsgPack(),  Id: utils.RandomString(12), ChainId: config.ChainId}
	pl.config = config
	if validator == nil {
		validator = &path.Validator
	}
	resp, err := (&pl).SendDataRequest(string(*validator))
	if err != nil {
		return nil, err
	}
	
	if resp == nil {
		return nil, apperror.Internal("timedout")
	}
	data, err := UnpackP2pEventResponse(resp.Data)
	
	if err != nil {
		return nil, err
	}
	if len(data.States) == 0 {
		return nil, apperror.NotFound("subnet not found")
	}
	return &data, encoder.MsgPackUnpackStruct(data.States[0], &result)
}

func GetEvent(config *configs.MainConfiguration, eventPath entities.EventPath, validator *entities.PublicKeyString) (*entities.Event, *P2pEventResponse, error) {
	pl := P2pPayload{Action: P2pActionGetEvent, Data: eventPath.MsgPack(),  Id: utils.RandomString(12), ChainId: config.ChainId}
	pl.config = config
	if validator == nil {
		validator = &eventPath.Validator
	}
	resp, err := (&pl).SendDataRequest(string(*validator))
	if err != nil {
		return nil, nil, err
	}
	if resp == nil {
		return nil, nil, apperror.Internal("timedout")
	}
	data, err := UnpackP2pEventResponse(resp.Data)
	
	if err != nil {
		logger.Errorf("UnpackError: %v", err)
		return nil, nil, err
	}
	if len(data.Event) == 0 {
		return nil, nil, apperror.NotFound("subnet not found")
	}
	event := entities.GetEventEntityFromModel(eventPath.Model)
	logger.Infof("EventMODELEData: %v", eventPath.Model)
	event.Payload = entities.ClientPayload{Data: event.Payload.Data}
	
	event, err = entities.UnpackEvent(data.Event, eventPath.Model)
	return event, &data, err
}

func (p *P2pPayload) Sign (privateKey []byte) (error) {
	p.Timestamp = uint64(time.Now().UnixMilli())
	b, err := p.EncodeBytes();
	if(err != nil) {
		return err
	}
	p.Signature, _  = crypto.SignEDD(b, privateKey)
	// singer := crypto.GetPublicKeyEDD([64]byte(privateKey))
    p.Signer = privateKey[32:]
	return nil
}


func UnpackP2pPayload(b []byte) (*P2pPayload, error) {
	var message  P2pPayload
	err := encoder.MsgPackUnpackStruct(b, &message)
	return &message, err
}
