package utils

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	// "math"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sirupsen/logrus"
)

const (
	PlatformEthereum string = "ethereum"
	PlatformBitcoin         = "bitcoin"
	PlatformSolana          = "solana"
	PlatformCosmos          = "cosmos"
)

/**
HANDSHAKE MESSAGE
**/
type HandshakeData struct {
	Timestamp  int    `json:"timestamp"`
	ProtocolId string `json:"protocolId"`
	Name       string `json:"name"`
	NodeType   uint   `json:"node_type"`
}

type Handshake struct {
	Data      HandshakeData `json:"data"`
	Signature string        `json:"signature"`
	Signer    string        `json:"signer"`
}

func (hs *Handshake) ToJSON() []byte {
	h, _ := json.Marshal(hs)
	return h
}
func (hs *Handshake) Init(jsonString string) error {
	er := json.Unmarshal([]byte(jsonString), &hs)
	return er
}
func (hsd *HandshakeData) ToString() string {
	return fmt.Sprintf("name:%s,timestamp:%d,protocolId:%s,nodeType:%d", hsd.Name, hsd.Timestamp, hsd.ProtocolId, hsd.NodeType)
}

func (hsd *HandshakeData) ToJSON() []byte {
	h, _ := json.Marshal(hsd)
	return h
}
func HandshakeFromJSON(json string) (Handshake, error) {
	data := Handshake{}
	er := data.Init(json)
	return data, er
}

func HandshakeFromBytes(b []byte) Handshake {
	var handshake Handshake
	if err := json.Unmarshal(b, &handshake); err != nil {
		panic(err)
	}
	return handshake
}

func HandshakeFromString(hs string) Handshake {
	return HandshakeFromBytes([]byte(hs))
}

func CreateHandshake(name string, network string, privateKey string, nodeType uint) Handshake {
	pubKey := GetPublicKey(privateKey)
	data := HandshakeData{Name: name, ProtocolId: network, NodeType: nodeType, Timestamp: int(time.Now().Unix())}
	_, signature := Sign((&data).ToString(), privateKey)
	return Handshake{Data: data, Signature: signature, Signer: pubKey}
}

/**
CHAT MESSAGE
**/
type ChatMessageHeader struct {
	Length    int    `json:"length"`
	Sender    string `json:"from"`
	Receiver  string `json:"reciever"`
	ChainId   string `json:"chainId"`
	Platform  string `json:"platform"`
	Timestamp uint   `json:"timestamp"`
}

// TODO! platform enum channel
//! receiver field is name of channel u are sending to
// ! look for all subscribers to the channel
//! channel subscribers store
type ChatMessageBody struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}
type ChatMessageAction struct {
	Contract   string   `json:"contract"`
	Abi        string   `json:"abi"`
	Action     string   `json:"action"`
	Parameters []string `json:"parameters"`
}

type ChatMessage struct {
	Header  ChatMessageHeader   `json:"header"`
	Body    ChatMessageBody     `json:"body"`
	Actions []ChatMessageAction `json:"actions"`
	Origin  string              `json:"origin"`
}

func (chatMessage *ChatMessage) ToString() string {
	values := []string{}
	// values = append(values, fmt.Sprintf("Header.Length:%d", chatMessage.Header.Length))
	values = append(values, fmt.Sprintf("Header.Sender:%s", chatMessage.Header.Sender))
	values = append(values, fmt.Sprintf("Header.Receiver:%s", chatMessage.Header.Receiver))
	values = append(values, fmt.Sprintf("Header.ChainId:%s", chatMessage.Header.ChainId))
	values = append(values, fmt.Sprintf("Header.Platform:%s", chatMessage.Header.Platform))
	values = append(values, fmt.Sprintf("Header.Timestamp:%d", chatMessage.Header.Timestamp))

	logger.WithFields(logrus.Fields{"subject: ": strings.ToLower(hexutil.Encode(Hash(chatMessage.Body.Subject)))}).Infof("subject")
	logger.WithFields(logrus.Fields{"message: ": strings.ToLower(hexutil.Encode(Hash(chatMessage.Body.Message)))}).Infof("message")
	values = append(values, fmt.Sprintf("Body.Subject:%s", strings.ToLower(hexutil.Encode(Hash(chatMessage.Body.Subject)))))
	values = append(values, fmt.Sprintf("Body.Message:%s", strings.ToLower(hexutil.Encode(Hash(chatMessage.Body.Message)))))
	_action := []string{}
	for i := 0; i < len(chatMessage.Actions); i++ {
		_action = append(_action, fmt.Sprintf("Actions[%d].Contract:%s", i, chatMessage.Actions[i].Contract))
		_action = append(_action, fmt.Sprintf("Actions[%d].Abi:%s", i, chatMessage.Actions[i].Abi))
		_action = append(_action, fmt.Sprintf("Actions[%d].Action:%s", i, chatMessage.Actions[i].Action))

		_parameter := []string{}
		for j := 0; j < len(chatMessage.Actions[i].Parameters); j++ {
			_parameter = append(_parameter, fmt.Sprintf("Actions[%d].Parameters[%d]:%s", i, j, chatMessage.Actions[i].Parameters[j]))
		}

		_action = append(_action, fmt.Sprintf("Actions[%d].Parameters:%s", i, _parameter))
	}

	values = append(values, fmt.Sprintf("Actions:%s", _action))
	values = append(values, fmt.Sprintf("Origin:%s", chatMessage.Origin))

	return strings.Join(values, ",")
}

/**
NODE MESSAGE
**/
type ClientMessage struct {
	Message         ChatMessage `json:"message"`
	SenderSignature string      `json:"senderSignature"`
	NodeSignature   string      `json:"nodeSignature"`
}

type SuccessResponse struct {
	body ClientMessage
	meta Meta
}

type ErrorResponse struct {
	statusCode int
	meta       Meta
}

type Meta struct {
	statusCode int
	success    bool
}

func ReturnError(msg string, code int) *ErrorResponse {
	meta := Meta{statusCode: code}
	meta.success = false
	e := ErrorResponse{statusCode: code}
	e.meta = meta
	return &e
}

func (msg *ClientMessage) ToJSON() []byte {
	m, _ := json.Marshal(msg)
	return m
}
func (msg *ClientMessage) ToString() string {
	return fmt.Sprintf("%s:%s:%s", msg.Message.ToString(), msg.SenderSignature, msg.NodeSignature)
}

func (msg *ClientMessage) Hash() []byte {
	bs := crypto.Keccak256Hash([]byte(msg.Message.ToString())).Bytes()
	return bs
}

func (msg *ClientMessage) Key() string {
	return fmt.Sprintf("/%s/%s", msg.Message.Header.Sender, hexutil.Encode(msg.Hash()))
}

func ClientMessageFromBytes(b []byte) (ClientMessage, error) {
	var message ClientMessage
	// if err := json.Unmarshal(b, &message); err != nil {
	// 	panic(err)
	// }
	err := json.Unmarshal(b, &message)
	return message, err
}

func ClientMessageFromString(msg string) (ClientMessage, error) {
	return ClientMessageFromBytes([]byte(msg))
}

func (msg *ChatMessage) ToJSON() string {
	m, _ := json.Marshal(msg)
	return string(m)
}

func ChatMessageFromBytes(b []byte) ChatMessage {
	var message ChatMessage
	if err := json.Unmarshal(b, &message); err != nil {
		panic(err)
	}
	return message
}

func ChatMessageFromString(msg string) ChatMessage {
	return ChatMessageFromBytes([]byte(msg))
}

type MessageJsonInput struct {
	Timestamp int                 `json:"timestamp"`
	From      string              `json:"from"`
	Receiver  string              `json:"receiver"`
	Platform  string              `json:"platform"`
	ChainId   string              `json:"chainId"`
	Type      string              `json:"type"`
	Message   string              `json:"message"`
	Subject   string              `json:"subject"`
	Signature string              `json:"signature"`
	Actions   []ChatMessageAction `json:"actions"`
	Origin    string              `json:"origin"`
}

func CreateMessageFromJson(msg MessageJsonInput) ChatMessage {

	chatMessage := ChatMessageHeader{
		Timestamp: uint(msg.Timestamp),
		Sender:    msg.From,
		Receiver:  msg.Receiver,
		ChainId:   msg.ChainId,
		Platform:  msg.Platform,
		Length:    100,
	}

	bodyMessage := ChatMessageBody{
		Subject: msg.Subject,
		Message: msg.Message,
	}
	logger.Infof("origin %s", msg.Origin)
	_chatMessage := ChatMessage{chatMessage, bodyMessage, msg.Actions, msg.Origin}
	return _chatMessage
}

func IsValidMessage(msg ChatMessage, signature string) bool {
	chatMessage := msg.ToJSON()
	signer := msg.Header.Sender
	if math.Abs(float64(int(msg.Header.Timestamp)/1000-int(time.Now().Unix()))) > VALID_HANDSHAKE_SECONDS {
		logger.WithFields(logrus.Fields{"data": chatMessage}).Warnf("ChatMessage Expired: %s", chatMessage)
		return false
	}
	message := msg.ToString()
	logger.Infof("message %s", message)
	isValid := VerifySignature(signer, message, signature)
	if !isValid {
		logger.WithFields(logrus.Fields{"message": message, "signature": signature}).Warnf("Invalid signer %s", signer)
		return false
	} else {

	}
	return true
}

// PubSubMessage
type PubSubMessage struct {
	Data      json.RawMessage `json:"data"`
	Timestamp string          `json:"timestamp"`
	Signature string          `json:"signature"`
}

func (msg *PubSubMessage) ToJSON() []byte {
	m, _ := json.Marshal(msg)
	return m
}

func (msg *PubSubMessage) ToString() string {
	values := []string{}
	values = append(values, fmt.Sprintf("Data:%s", string(msg.Data)))
	values = append(values, fmt.Sprintf("Timestmap%s", msg.Timestamp))
	return strings.Join(values, ",")
}

func NewSignedPubSubMessage(data []byte, privateKey string) PubSubMessage {
	timestamp := int(time.Now().Unix())
	message := PubSubMessage{Data: data, Timestamp: strconv.Itoa(timestamp)}
	_, sig := Sign(message.ToString(), privateKey)
	message.Signature = sig
	return message
}

func PubSubMessageFromBytes(b []byte) (PubSubMessage, error) {
	var message PubSubMessage
	err := json.Unmarshal(b, &message)
	return message, err
}

// Batch
type Batch struct {
	BatchId    string `json:"batchId"`
	Size       int    `json:"size"`
	Closed     bool   `json:"closed"`
	NodeHeight int    `json:"nodeHeight"`
	Hash       string `json:"hash"`
	Timestamp  int    `json:"timestamp"`
}

func (msg *Batch) ToJSON() []byte {
	m, _ := json.Marshal(msg)
	return m
}

func (msg *Batch) ToString() string {
	values := []string{}
	values = append(values, fmt.Sprintf("BatchId:%s", string(msg.BatchId)))
	values = append(values, fmt.Sprintf("Size:%s", strconv.Itoa(msg.Size)))
	values = append(values, fmt.Sprintf("NodeHeight:%s", strconv.Itoa(msg.NodeHeight)))
	values = append(values, fmt.Sprintf("Timestamp:%s", strconv.Itoa(msg.Timestamp)))
	values = append(values, fmt.Sprintf("Hash:%s", msg.Hash))
	return strings.Join(values, ",")
}

func (msg *Batch) Key() string {
	return fmt.Sprintf("/%s", msg.BatchId)
}

// func (msg *Batch) Sign(privateKey string) Batch {

// 	msg.Timestamp = int(time.Now().Unix())
// 	_, sig := Sign(msg.ToString(), privateKey)
// 	msg.Signature = sig
// 	return *msg
// }

func NewBatch() Batch {
	id, _ := gonanoid.New()
	return Batch{BatchId: id,
		Size:   0,
		Closed: false}
}

func BatchFromBytes(b []byte) (Batch, error) {
	var message Batch
	err := json.Unmarshal(b, &message)
	return message, err
}

// DeliveryProof
type DeliveryProof struct {
	MessageHash   string `json:"messageHash"`
	MessageSender string `json:"messageSender"`
	NodeAddress   string `json:"node"`
	Timestamp     int    `json:"timestamp"`
	Signature     string `json:"signature"`
	Batch         string `json:"batch"`
	Index         int    `json:"index"`
}

func (msg *DeliveryProof) ToJSON() []byte {
	m, _ := json.Marshal(msg)
	return m
}

func (msg *DeliveryProof) Key() string {
	return fmt.Sprintf("/%s/%s", msg.MessageHash, msg.MessageSender)
}
func (msg *DeliveryProof) BatchKey() string {
	return fmt.Sprintf("/%s", msg.Batch)
}

func (msg *DeliveryProof) ToString() string {
	values := []string{}
	values = append(values, fmt.Sprintf("Message:%s", string(msg.MessageHash)))
	values = append(values, fmt.Sprintf("NodeAddress:%s", msg.NodeAddress))
	values = append(values, fmt.Sprintf("Timestamp:%s", strconv.Itoa(msg.Timestamp)))
	return strings.Join(values, ",")
}

// func NewSignedDeliveryProof(data []byte, privateKey string) DeliveryProof {
// 	message, _ := DeliveryProofFromBytes(data)
// 	_, sig := Sign(message.ToString(), privateKey)
// 	message.Signature = sig
// 	return message
// }

func DeliveryProofFromBytes(b []byte) (DeliveryProof, error) {
	var message DeliveryProof
	err := json.Unmarshal(b, &message)
	return message, err
}

// DeliveryClaim
type DeliveryClaim struct {
	NodeHeight int      `json:"nodeHeight"`
	Signature  string   `json:"signature"`
	Amount     string   `json:"amount"`
	Proofs     []string `json:"proofs"`
}

func (msg *DeliveryClaim) ToJSON() []byte {
	m, _ := json.Marshal(msg)
	return m
}

func DeliveryClaimFromBytes(b []byte) (DeliveryClaim, error) {
	var message DeliveryClaim
	err := json.Unmarshal(b, &message)
	return message, err
}
