package keeper // noalias

import (
	"github.com/cosmos/cosmos-sdk/collections"
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// does a certain by-power index record exist
func ValidatorByPowerIndexExists(ctx sdk.Context, keeper Keeper, power []byte) bool {
	store := ctx.KVStore(keeper.storeKey)
	return store.Has(power)
}

// update validator for testing
func TestingUpdateValidator(keeper Keeper, ctx sdk.Context, validator types.Validator, apply bool) types.Validator {
	keeper.Validators.Insert(ctx, validator.GetOperator(), validator)

	if !apply {
		ctx, _ = ctx.CacheContext()
	}
	_, err := keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	if err != nil {
		panic(err)
	}

	validator, found := keeper.GetValidator(ctx, validator.GetOperator())
	if !found {
		panic("validator expected but not found")
	}

	return validator
}

// RandomValidator returns a random validator given access to the keeper and ctx
func RandomValidator(r *rand.Rand, keeper Keeper, ctx sdk.Context) (val types.Validator, ok bool) {
	vals := keeper.Validators.Iterate(ctx, collections.Range[sdk.ValAddress]{}).Values()
	if len(vals) == 0 {
		return types.Validator{}, false
	}

	i := r.Intn(len(vals))

	return vals[i], true
}
