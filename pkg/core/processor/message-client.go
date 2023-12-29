package processor

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ipfs/go-datastore/query"
	"github.com/mlayerprotocol/go-mlayer/channelpool"
	"github.com/mlayerprotocol/go-mlayer/entities"
	db "github.com/mlayerprotocol/go-mlayer/pkg/core/db"
	"github.com/mlayerprotocol/go-mlayer/utils"
)

var logger = &utils.Logger

func ValidateMessageClient(
	ctx context.Context,
	connectedSubscribers *map[string]map[string][]interface{},
	clientHandshake *utils.ClientHandshake,
	channelSubscriberStore *db.Datastore) {
	// VALIDATE AND DISTRIBUTE
	utils.Logger.Infof("Signer:  %s\n", clientHandshake.Signer)
	results, err := channelSubscriberStore.Query(ctx, query.Query{
		Prefix: "/" + clientHandshake.Signer,
	})
	if err != nil {
		utils.Logger.Errorf("Channel Subscriber Store Query Error %o", err)
		return
	}
	entries, _err := results.Rest()
	for i := 0; i < len(entries); i++ {
		_sub, _ := entities.SubscriptionFromBytes(entries[i].Value)
		if (*connectedSubscribers)[_sub.ChannelId] == nil {
			(*connectedSubscribers)[_sub.ChannelId] = make(map[string][]interface{})
		}
		(*connectedSubscribers)[_sub.ChannelId][_sub.Subscriber] = append((*connectedSubscribers)[_sub.ChannelId][_sub.Subscriber], clientHandshake.ClientSocket)
	}
	utils.Logger.Infof("results:  %s  -  %o\n", entries[0].Value, _err)
}

func ValidateAndAddToDeliveryProofToBlock(ctx context.Context,
	proof *entities.DeliveryProof,
	deliveryProofStore *db.Datastore,
	channelSubscriberStore *db.Datastore,
	stateStore *db.Datastore,
	localBlockStore *db.Datastore,
	MaxBlockSize int,
	mutex *sync.RWMutex,
) {
	err := deliveryProofStore.Set(ctx, db.Key(proof.Key()), proof.Pack(), true)
	if err == nil {
		// msg, err := validMessagesStore.Get(ctx, db.Key(fmt.Sprintf("/%s/%s", proof.MessageSender, proof.MessageHash)))
		// if err != nil {
		// 	// invalid proof or proof has been tampered with
		// 	return
		// }
		// get signer of proof
		susbscriber, err := utils.GetSigner(proof.ToString(), proof.Signature)
		if err != nil {
			// invalid proof or proof has been tampered with
			return
		}
		// check if the signer of the proof is a member of the channel
		isSubscriber, err := channelSubscriberStore.Has(ctx, db.Key("/"+susbscriber+"/"+proof.MessageHash))
		if isSubscriber {
			// proof is valid, so we should add to a new or existing batch
			var block *entities.Block
			var err error
			txn, err := stateStore.NewTransaction(ctx, false)
			if err != nil {
				utils.Logger.Errorf("State query errror %o", err)
				// invalid proof or proof has been tampered with
				return
			}
			blockData, err := txn.Get(ctx, db.Key(utils.CurrentDeliveryProofBlockStateKey))
			if err != nil {
				logger.Errorf("State query errror %o", err)
				// invalid proof or proof has been tampered with
				txn.Discard(ctx)
				return
			}
			if len(blockData) > 0 && block.Size < MaxBlockSize {
				block, err = entities.UnpackBlock(blockData)
				if err != nil {
					logger.Errorf("Invalid batch %o", err)
					// invalid proof or proof has been tampered with
					txn.Discard(ctx)
					return
				}
			} else {
				// generate a new batch
				block = entities.NewBlock()

			}
			block.Size += 1
			if block.Size >= MaxBlockSize {
				block.Closed = true
				block.NodeHeight = utils.GetNodeHeight()
			}
			// save the proof and the batch
			block.Hash = hexutil.Encode(utils.Hash(proof.Signature + block.Hash))
			err = txn.Put(ctx, db.Key(utils.CurrentDeliveryProofBlockStateKey), block.Pack())
			if err != nil {
				logger.Errorf("Unable to update State store error %o", err)
				txn.Discard(ctx)
				return
			}
			proof.Block = block.BlockId
			proof.Index = block.Size
			err = deliveryProofStore.Put(ctx, db.Key(proof.Key()), proof.Pack())
			if err != nil {
				txn.Discard(ctx)
				logger.Errorf("Unable to save proof to store error %o", err)
				return
			}
			err = localBlockStore.Put(ctx, db.Key(utils.CurrentDeliveryProofBlockStateKey), block.Pack())
			if err != nil {
				logger.Errorf("Unable to save batch error %o", err)
				txn.Discard(ctx)
				return
			}
			err = txn.Commit(ctx)
			if err != nil {
				logger.Errorf("Unable to commit state update transaction errror %o", err)
				txn.Discard(ctx)
				return
			}
			// dispatch the proof and the batch
			if block.Closed {
				channelpool.OutgoingDeliveryProof_BlockC <- block
			}
			channelpool.OutgoingDeliveryProofC <- proof

		}

	}

}
