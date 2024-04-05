// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package registry

import (
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/sausaging/hypersdk/chain"
	"github.com/sausaging/hypersdk/codec"

	"github.com/sausaging/hypersdk/x/programs/cmd/simulator/vm/actions"
	"github.com/sausaging/hypersdk/x/programs/cmd/simulator/vm/consts"
)

// Setup types
func init() {
	consts.ActionRegistry = codec.NewTypeParser[chain.Action, *warp.Message]()
	consts.AuthRegistry = codec.NewTypeParser[chain.Auth, *warp.Message]()

	errs := &wrappers.Errs{}
	errs.Add(
		// When registering new actions, ALWAYS make sure to append at the end.
		consts.ActionRegistry.Register((&actions.ProgramCreate{}).GetTypeID(), actions.UnmarshalProgramCreate, false),
		consts.ActionRegistry.Register((&actions.ProgramExecute{}).GetTypeID(), actions.UnmarshalProgramExecute, false),

		// When registering new auth, ALWAYS make sure to append at the end.
	)
	if errs.Errored() {
		panic(errs.Err)
	}
}
