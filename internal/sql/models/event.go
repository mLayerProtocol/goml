package models

import (
	"github.com/mlayerprotocol/go-mlayer/entities"
)

type EventInterface interface {
}

type DeleteEvent struct {
	entities.Event `msgpack:",noinline"`
	//IsValid   bool `gorm:"default:false" json:"isVal"`
	EventType int16 `json:"t"`
	// Payload datatypes.JSON  `json:"pld"`
}


func GetStateModelFromModelType(modelType entities.EntityModel) (any) {
	var table any
	switch modelType  {
	case entities.TopicModel:
		table = TopicState{}
	case entities.AuthModel:
		table = AuthorizationState{}
	case entities.SubnetModel:
		table = SubnetState{}
	case entities.SubscriptionModel:
		table = SubnetState{}
	case entities.WalletModel:
		table = WalletState{}
	}
	return table
}
func GetEventModelFromModelType(modelType entities.EntityModel) (any) {
	var table any
	switch modelType  {
	case entities.TopicModel:
		table = TopicEvent{}
	case entities.AuthModel:
		table = AuthorizationEvent{}
	case entities.SubnetModel:
		table = SubnetEvent{}
	case entities.SubscriptionModel:
		table = SubscriptionEvent{}
	case entities.WalletModel:
		table = WalletEvent{}
	}
	return table
}