// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"context"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/version"
)

type BroadCastTreeHandler struct {
	vm *VM
}

func NewBroadCastTreeHandler(vm *VM) *BroadCastTreeHandler {
	return &BroadCastTreeHandler{vm}
}

func (b *BroadCastTreeHandler) Connected(
	ctx context.Context,
	nodeID ids.NodeID,
	v *version.Application,
) error {
	return nil
}

func (b *BroadCastTreeHandler) Disconnected(ctx context.Context, nodeID ids.NodeID) error {
	return nil
}

func (*BroadCastTreeHandler) AppGossip(context.Context, ids.NodeID, []byte) error {
	return nil
}

func (b *BroadCastTreeHandler) AppRequest(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
	_ time.Time,
	request []byte,
) error {

	return b.vm.broadcastManager.AppRequest(ctx, nodeID, requestID, request)
}

func (b *BroadCastTreeHandler) AppRequestFailed(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
) error {
	return b.vm.broadcastManager.HandleRequestFailed(requestID)
}

func (b *BroadCastTreeHandler) AppResponse(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
	response []byte,
) error {
	return b.vm.broadcastManager.HandleResponse(requestID, response)
}

func (*BroadCastTreeHandler) CrossChainAppRequest(
	context.Context,
	ids.ID,
	uint32,
	time.Time,
	[]byte,
) error {
	return nil
}

func (*BroadCastTreeHandler) CrossChainAppRequestFailed(context.Context, ids.ID, uint32) error {
	return nil
}

func (*BroadCastTreeHandler) CrossChainAppResponse(context.Context, ids.ID, uint32, []byte) error {
	return nil
}
