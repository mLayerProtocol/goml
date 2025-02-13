package service

import (
	"context"
	"encoding/hex"

	"github.com/ipfs/go-datastore"
	"github.com/mlayerprotocol/go-mlayer/common/apperror"
	"github.com/mlayerprotocol/go-mlayer/common/constants"
	"github.com/mlayerprotocol/go-mlayer/common/utils"
	"github.com/mlayerprotocol/go-mlayer/configs"
	"github.com/mlayerprotocol/go-mlayer/entities"
	dsquery "github.com/mlayerprotocol/go-mlayer/internal/ds/query"
	"github.com/mlayerprotocol/go-mlayer/internal/sql/models"
	"gorm.io/gorm"
)

/*
Validate an agent authorization
*/
func ValidateTopicData(topic *entities.Topic, authState *models.AuthorizationState) (currentTopicState *models.TopicState, err error) {

	// TODO state might have changed befor receiving event, so we need to find state that is relevant to this event.

	if topic.ID != "" {
		_topicState, err := dsquery.GetTopicById(topic.ID)
		if err != nil && !dsquery.IsErrorNotFound(err) {
			if err == gorm.ErrRecordNotFound {
				return nil, apperror.Forbidden("Invalid subnet id")
			}
		}
		if _topicState != nil {
			currentTopicState = &models.TopicState{Topic: *_topicState}
		}
	}
	if authState != nil && *authState.Priviledge < constants.MemberPriviledge {
		return nil, apperror.Forbidden("Agent does not have enough permission to create topics")
	}

	if len(topic.Ref) > 40 {
		return nil, apperror.BadRequest("Topic reference can not be more than 40 characters")
	}
	if !utils.IsAlphaNumericDot(topic.Ref) {
		return nil, apperror.BadRequest("Reference must be alphanumeric, _ and . but cannot start with a number")
	}
	return currentTopicState, nil

}

func saveTopicEvent(where entities.Event, createData *entities.Event, updateData *entities.Event, txn *datastore.Txn, tx *gorm.DB) (*entities.Event, error) {
	return SaveEvent(entities.TopicModel, where, createData, updateData, txn)
}

func HandleNewPubSubTopicEvent(event *entities.Event, ctx *context.Context) error {
	cfg, ok := (*ctx).Value(constants.ConfigKey).(*configs.MainConfiguration)
	if !ok {
		panic("Unable to get config from context")
	}

	dataStates := dsquery.NewDataStates(cfg)
	dataStates.AddEvent(*event)
	
	data := event.Payload.Data.(entities.Topic)

	var id = data.ID
	if len(data.ID) == 0 {
		id, _ = entities.GetId(data, data.ID)
	} else {
		id = data.ID
	}
	data.Event = *event.GetPath()
	data.BlockNumber = event.BlockNumber
	data.Cycle = event.Cycle
	data.Epoch = event.Epoch
	data.EventSignature = event.Signature
	hash, err := data.GetHash()
	if err != nil {
		return err
	}
	data.Hash = hex.EncodeToString(hash)
	data.Account = event.Payload.Account
	data.Agent = event.Payload.Agent
	data.Timestamp = event.Payload.Timestamp
	logger.Debug("Processing 1...")
	var localState models.TopicState

	validator := utils.IfThenElse(event.IsLocal(cfg),  "",  string(event.Validator))
	
	defer func () {
		stateUpdateError := dataStates.Commit(nil, nil, nil)
		if stateUpdateError != nil {
			
			panic(stateUpdateError)
		} else {
			go  OnFinishProcessingEvent(ctx, event,  &data)
			// go utils.WriteBytesToFile(filepath.Join(cfg.DataDir, "log.txt"), []byte("newMessage" + "\n"))
		}	
	}()
	// err := query.GetOne(&models.TopicState{Topic: entities.Topic{ID: id}}, &localState)
	// err = sql.SqlDb.Where(&models.TopicState{Topic: entities.Topic{ID: id}}).Take(&localState).Error
	// topic :=  &entities.Topic{}
	if data.ID != "" {
		topic :=  &entities.Topic{}
	 //topic, err := dsquery.GetTopicById(id)
		_, err = SyncTypedStateById(id, topic, cfg, validator)
		if err != nil {
			logger.Error("HandleNewPubSubAuthEvent/SyncTypedStateById", err)
			return err
		}
		localState = models.TopicState{Topic: *topic}

	}
	logger.Debug("Processing 2...")
	// stateTxn, err := stores.StateStore.NewTransaction(context.Background(), false) // true for read-write, false for read-only
	// if err != nil {
	// 	// either subnet does not exist or you are not uptodate
	// }
	// txn, err := stores.EventStore.NewTransaction(context.Background(), false) // true for read-write, false for read-only
	// if err != nil {
	// 	// either subnet does not exist or you are not uptodate
	// }
	// defer stateTxn.Discard(context.Background())
	// defer txn.Discard(context.Background())

	var localDataState *LocalDataState
	if localState.ID != "" {
		localDataState = &LocalDataState{
			ID:        localState.ID,
			Hash:      localState.ID,
			Event:     &localState.Event,
			Timestamp: localState.Timestamp,
		}
	}
	// localDataState := utils.IfThenElse(localState != nil, &LocalDataState{
	// 	ID: localState.ID,
	// 	Hash: localState.ID,
	// 	Event: &localState.Event,
	// 	Timestamp: localState.Timestamp,
	// }, nil)
	var stateEvent *entities.Event
	if localState.ID != "" {
		stateEvent, err = dsquery.GetEventFromPath(&localState.Event)
		if err != nil && !dsquery.IsErrorNotFound(err) {
		
			logger.Error("HandleNewPubSubAuthEvent/GetEventFromPath", err)
			return err
		}
	}

	var localDataStateEvent *LocalDataStateEvent
	if stateEvent != nil {
		localDataStateEvent = &LocalDataStateEvent{
			ID:        stateEvent.ID,
			Hash:      stateEvent.Hash,
			Timestamp: stateEvent.Timestamp,
		}
	}

	eventData := PayloadData{Subnet: data.Subnet, localDataState: localDataState, localDataStateEvent: localDataStateEvent}
	// tx := sql.SqlDb
	// // defer func () {
	// 	if tx.Error != nil {
	// 		tx.Rollback()
	// 	} else {
	// 		tx.Commit()
	// 	}
	// }()

	previousEventUptoDate, authEventUptoDate, authState, eventIsMoreRecent, err := ProcessEvent(event, eventData, true, saveTopicEvent,nil, nil, ctx, dataStates)
	if err != nil {
		logger.Error("ProcessEventError ", err)
		return err
	}
	// err = dsquery.IncrementCounters(event.Cycle, event.Validator, event.Subnet, &txn)
	// if err != nil { 
	// 	logger.Errorf("ErorrIncrementingCounters: %v", err)
	// 	return err
	// }
	logger.Infof("SuccessfullyIncrementingCounters")
	if previousEventUptoDate && authEventUptoDate {
		if !event.IsLocal(cfg) {
			_, err = ValidateTopicData(&data, authState)
		}
		
		if err != nil {
			// update error and mark as synced
			// notify validator of error
			// saveTopicEvent(entities.Event{ID: event.ID}, nil, &entities.Event{Error: err.Error(), IsValid: utils.FalsePtr(), Synced: utils.TruePtr()}, &txn, nil)
			dataStates.AddEvent(entities.Event{ID: event.ID, Error: err.Error(), IsValid: utils.FalsePtr(), Synced:  utils.TruePtr()})
			logger.Infof("ERROEVENT, %+v ==>",err.Error())
		} else {
			// TODO if event is older than our state, just save it and mark it as synced
			// _, err := saveTopicEvent(entities.Event{ID: event.ID}, nil, &entities.Event{IsValid: utils.TruePtr(), Synced: utils.TruePtr()}, &txn, nil)
			dataStates.AddEvent(entities.Event{ID: event.ID, IsValid:  utils.TruePtr(), Synced:  utils.TruePtr()})
			data.ID, _ = entities.GetId(data, data.ID)
			logger.Infof("TOPICSTATE, %+v ==>",data.ID)
			if eventIsMoreRecent {
				// update state
					dataStates.AddCurrentState(entities.TopicModel,id, data)	
			} else {
				dataStates.AddHistoricState(entities.TopicModel,id, data.MsgPack())
			}
			
		}

	}
	return nil
}
