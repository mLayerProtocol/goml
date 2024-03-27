package client

import (
	// "errors"

	"encoding/json"

	"github.com/mlayerprotocol/go-mlayer/common/apperror"
	"github.com/mlayerprotocol/go-mlayer/common/constants"
	"github.com/mlayerprotocol/go-mlayer/entities"
	"github.com/mlayerprotocol/go-mlayer/internal/service"
	"github.com/mlayerprotocol/go-mlayer/internal/sql/models"
	query "github.com/mlayerprotocol/go-mlayer/internal/sql/query"
	"gorm.io/gorm"
)

func GetSubscriptions() (*[]models.SubscriptionState, error) {
	var subscriptionStates []models.SubscriptionState

	err := query.GetMany(models.SubscriptionState{}, &subscriptionStates)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &subscriptionStates, nil
}

func GetAccountSubscriptions() (*[]models.TopicState, error) {

	// query.GetAccountSubscriptions("did:cosmos1vxm0v5dm9hacm3mznvx852fmtu6792wpa4wgqx")
	var subscriptionStates []models.SubscriptionState
	var topicStates []models.TopicState

	err := query.GetMany(models.SubscriptionState{Subscription: entities.Subscription{Subscriber: "did:cosmos1vxm0v5dm9hacm3mznvx852fmtu6792wpa4wgqx"}}, &subscriptionStates)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	topicIds := make([]string, len(subscriptionStates))

	for _, sub := range subscriptionStates {
		topicIds = append(topicIds, sub.Topic)
	}

	topErr := query.GetWithIN(models.TopicState{}, &topicStates, topicIds)
	if topErr != nil {
		if topErr == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &topicStates, nil
}

func GetSubscription(id string) (*models.SubscriptionState, error) {
	subscriptionState := models.SubscriptionState{}

	err := query.GetOne(models.SubscriptionState{
		Subscription: entities.Subscription{ID: id},
	}, &subscriptionState)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &subscriptionState, nil

}

func ValidateSubscriptionPayload(payload entities.ClientPayload, authState *models.AuthorizationState) (
	assocPrevEvent *entities.EventPath,
	assocAuthEvent *entities.EventPath,
	err error,
) {

	payloadData := entities.Subscription{}
	d, _ := json.Marshal(payload.Data)
	e := json.Unmarshal(d, &payloadData)
	if e != nil {
		logger.Errorf("UnmarshalError %v", e)
	}
	// if authState == nil {
	// 	return nil, nil,  apperror.Unauthorized("Agent not authorized")
	// }
	// if authState.Priviledge != constants.AdminPriviledge {
	// 	return nil, nil,  apperror.Unauthorized("Agent does not have sufficient privileges to perform this action")
	// }
	payload.Data = payloadData

	topicData, err := GetTopicById(payloadData.Topic)
	if err != nil {
		return nil, nil, err
	}

	if topicData == nil {
		return nil, nil, apperror.BadRequest("Invalid topic")
	}

	// pool = channelpool.SubscriptionEventPublishC

	currentState, err := service.ValidateSubscriptionData(&payloadData, &payload)
	if err != nil && (err != gorm.ErrRecordNotFound && payload.EventType == uint16(constants.SubscribeTopicEvent)) {
		return nil, nil, err
	}

	if currentState == nil && payload.EventType != uint16(constants.SubscribeTopicEvent) {
		return nil, nil, apperror.BadRequest("Account not subscribed")
	}
	if currentState != nil && currentState.Status != 0 && payload.EventType == uint16(constants.SubscribeTopicEvent) {
		return nil, nil, apperror.BadRequest("Account already subscribed")
	}

	// generate associations
	if currentState != nil {
		assocPrevEvent = entities.NewEventPath(entities.SubscriptionEventModel, currentState.EventHash)
	} else {
		assocPrevEvent = entities.NewEventPath(entities.TopicEventModel, topicData.EventHash)
	}

	if authState != nil {
		assocAuthEvent = entities.NewEventPath(entities.AuthEventModel, authState.EventHash)
	}
	return assocPrevEvent, assocAuthEvent, nil
}
