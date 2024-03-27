package vm

import (
	"context"
	"encoding/binary"
	"strconv"
	"sync"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/utils/set"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/filedb"
	"github.com/ava-labs/hypersdk/heap"
	"github.com/ava-labs/hypersdk/utils"
	"go.uber.org/zap"
)

const (
	deployPrefix uint8 = 0x7
)

type BroadCastManager struct {
	vm        *VM
	appSender common.AppSender
	fileDB    *filedb.FileDB
	l         sync.Mutex
	requestID uint32

	pendingJobs *heap.Heap[*broadcastJob, int64]
	jobs        map[uint32]*broadcastJob
	done        chan struct{}
}

type broadcastJob struct {
	nodeID       ids.NodeID
	imageID      ids.ID
	proofValType uint16
	chunkIndex   uint16
	data         []byte
}

func NewBroadCastManager(vm *VM) *BroadCastManager {
	return &BroadCastManager{
		vm:          vm,
		fileDB:      vm.fileDB,
		pendingJobs: heap.New[*broadcastJob, int64](64, true),
		jobs:        make(map[uint32]*broadcastJob),
		done:        make(chan struct{}),
	}
}

func (b *BroadCastManager) Run(appSender common.AppSender) {
	b.appSender = appSender
	b.vm.Logger().Info("starting broadcast manager")
	defer close(b.done)

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			b.l.Lock()
			for b.pendingJobs.Len() > 0 && len(b.jobs) < maxOutstanding {
				first := b.pendingJobs.First()
				b.pendingJobs.Pop()
				// send request
				job := first.Item
				if err := b.request(context.Background(), job); err != nil {
					b.vm.Logger().Error("unable to broadcast message", zap.Stringer("nodeID", job.nodeID), zap.Error(err))
				}
			}
			l := b.pendingJobs.Len()
			b.l.Unlock()
			b.vm.snowCtx.Log.Debug("checked for ready jobs", zap.Int("pending", l))
		case <-b.vm.stop:
			b.vm.Logger().Info("stopping broadcast manager")
			return
		}
	}
}

// you must hold [b.l] when calling this function
func (b *BroadCastManager) request(
	ctx context.Context,
	j *broadcastJob,
) error {
	requestID := b.requestID
	b.requestID++
	b.jobs[requestID] = j
	initial := consts.Uint64Len*2 + consts.IDLen + len(j.data)
	// pack the data
	p := codec.NewWriter(initial, initial*2)
	p.PackID(j.imageID)
	p.PackUint64(uint64(j.proofValType))
	p.PackUint64(uint64(j.chunkIndex))
	p.PackBytes(j.data)
	return b.appSender.SendAppRequest(
		ctx,
		set.Of(j.nodeID),
		requestID,
		p.Bytes(),
	)
}

func (b *BroadCastManager) AppRequest(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
	request []byte,
) error {
	r := codec.NewReader(request, consts.MaxInt)
	var imageID ids.ID
	r.UnpackID(true, &imageID)
	proofValType := r.UnpackUint64(true)
	chunkIndex := r.UnpackUint64(true)
	var data []byte
	r.UnpackBytes(-1, true, &data)
	if err := r.Err(); err != nil {
		b.vm.snowCtx.Log.Warn("unable to unpack request", zap.Error(err))
	}
	b.vm.snowCtx.Log.Info("received broadcast message", zap.Stringer("nodeID", nodeID), zap.Stringer("imageID: ", imageID), zap.Uint64("proofValType: ", proofValType), zap.Uint64("chunk index: ", chunkIndex))
	//============================================== convert to a function
	// @todo introduce functionality based on chunk index
	key := b.DeployKey(imageID, uint16(proofValType))
	has, err := b.fileDB.Has(key)
	if err != nil {
		b.vm.snowCtx.Log.Error("unable to check if file exists", zap.Error(err))
		return nil
	}
	if has {
		dataF, err := b.fileDB.Get(key)
		if err != nil { // this should not happen as file existance has been checked right now
			b.vm.snowCtx.Log.Error("unable to get data from file db", zap.Error(err))
		}
		data2 := make([]byte, len(data))
		copy(data2, data)
		data2 = append(dataF, data2...)
		err = b.fileDB.Put(key, data2)
		if err != nil {
			b.vm.snowCtx.Log.Error("unable to put data in file db", zap.Error(err))
		}
	} else {
		err = b.fileDB.Put(key, data)
		if err != nil {
			b.vm.snowCtx.Log.Error("unable to put data in file db", zap.Error(err))
		}
	}
	//==============================================
	return b.appSender.SendAppResponse(ctx, nodeID, requestID, nil)
}

func (b *BroadCastManager) HandleResponse(requestID uint32, response []byte) error {
	b.l.Lock()
	_, ok := b.jobs[requestID]
	delete(b.jobs, requestID)
	b.l.Unlock()

	if !ok {
		return nil
	}

	b.vm.snowCtx.Log.Info("received response", zap.Uint32("requestID", requestID), zap.Int("responseLen", len(response)))
	return nil
}

func (b *BroadCastManager) HandleRequestFailed(requestID uint32) error {
	b.l.Lock() //@todo is this the better way to handle error
	_, ok := b.jobs[requestID]
	delete(b.jobs, requestID)
	b.l.Unlock()
	if !ok {
		return nil
	}

	b.vm.snowCtx.Log.Info("request failed", zap.Uint32("requestID", requestID))
	return nil

}
func (b *BroadCastManager) Done() {
	<-b.done
}

func (b *BroadCastManager) Broadcast(
	ctx context.Context,
	imageID ids.ID,
	proofValType uint16,
	chunkIndex uint16,
	data []byte,
) {
	//============================================== convert to a function
	// @todo introduce functionality based on chunk index
	key := b.DeployKey(imageID, uint16(proofValType))
	has, err := b.fileDB.Has(key)
	if err != nil {
		b.vm.snowCtx.Log.Error("unable to check if file exists", zap.Error(err))
		return
	}
	if has {
		dataF, err := b.fileDB.Get(key)
		if err != nil { // this should not happen as file existance has been checked right now
			b.vm.snowCtx.Log.Error("unable to get data from file db", zap.Error(err))
		}
		data2 := make([]byte, len(data))
		copy(data2, data)
		data2 = append(dataF, data2...)
		err = b.fileDB.Put(key, data2)
		if err != nil {
			b.vm.snowCtx.Log.Error("unable to put data in file db", zap.Error(err))
		}
	} else {
		err = b.fileDB.Put(key, data)
		if err != nil {
			b.vm.snowCtx.Log.Error("unable to put data in file db", zap.Error(err))
		}
	}
	//==============================================
	height, err := b.vm.snowCtx.ValidatorState.GetCurrentHeight(ctx)
	if err != nil {
		b.vm.snowCtx.Log.Error("unable to get current p-chain height", zap.Error(err))
		return
	}
	validators, err := b.vm.snowCtx.ValidatorState.GetValidatorSet(
		ctx,
		height,
		b.vm.snowCtx.SubnetID,
	)
	if err != nil {
		b.vm.snowCtx.Log.Error("unable to get validator set", zap.Error(err))
		return
	}

	for nodeID := range validators {

		if nodeID == b.vm.snowCtx.NodeID {
			// don't send proofs to same node again.
			continue
		}

		idb := make([]byte, consts.IDLen+consts.Uint16Len*2+consts.NodeIDLen)
		copy(idb, imageID[:])
		binary.BigEndian.PutUint16(idb[consts.IDLen:], proofValType)
		binary.BigEndian.PutUint16(idb[consts.IDLen+consts.Uint16Len:], chunkIndex)
		copy(idb[consts.IDLen+consts.Uint16Len*2:], nodeID.Bytes())
		id := utils.ToID(idb)
		b.vm.snowCtx.Log.Info("broadcasting to node", zap.String("nodeID", nodeID.String()), zap.String("id", id.String()))
		b.l.Lock()
		b.pendingJobs.Push(&heap.Entry[*broadcastJob, int64]{
			ID: id,
			Item: &broadcastJob{
				nodeID:       nodeID,
				imageID:      imageID,
				proofValType: proofValType,
				chunkIndex:   chunkIndex,
				data:         data,
			},
			Index: b.pendingJobs.Len(),
		})
		b.l.Unlock()
	}
}

func (b *BroadCastManager) DeployKey(
	imageID ids.ID,
	proofValType uint16,
) string {
	return strconv.Itoa(int(deployPrefix)) + imageID.Hex() + strconv.Itoa(int(proofValType)) + ".pvalt"
}
