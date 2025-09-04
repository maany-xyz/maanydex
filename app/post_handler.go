package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PostHandlerOptions are the options required for constructing a FeeMarket PostHandler.
type PostHandlerOptions struct{}


// NewPostHandler returns a PostHandler chain with the fee deduct decorator.
func NewPostHandler(options PostHandlerOptions) (sdk.PostHandler, error) {
	var decorators []sdk.PostDecorator
  // (add post decorators here later if you need them)
	return sdk.ChainPostDecorators(decorators...), nil
}
