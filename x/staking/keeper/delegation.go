package keeper

import (
	"bytes"
	"fmt"
	"github.com/cosmos/cosmos-sdk/collections"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// GetDelegation returns a specific delegation.
func (k Keeper) GetDelegation(ctx sdk.Context,
	delAddr sdk.AccAddress, valAddr sdk.ValAddress,
) (delegation types.Delegation, found bool) {
	delegation, err := k.Delegations.Get(ctx, collections.Join(delAddr, valAddr))
	return delegation, err == nil
}

// GetValidatorDelegations returns all delegations to a specific validator.
// Useful for querier.
func (k Keeper) GetValidatorDelegations(ctx sdk.Context, valAddr sdk.ValAddress) (delegations []types.Delegation) { //nolint:interfacer
	// in theory could be an index in indexedmap, but we dont wanna be state machine breaking
	// is this even efficient?? iterating all delegations??
	iter := k.Delegations.Iterate(ctx, collections.PairRange[sdk.AccAddress, sdk.ValAddress]{})
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		if !iter.Key().K2().Equals(valAddr) {
			continue
		}
		delegations = append(delegations, iter.Value())
	}
	return delegations
}

// RemoveDelegation removes a delegation.
func (k Keeper) RemoveDelegation(ctx sdk.Context, delegation types.Delegation) {
	delegatorAddress := sdk.MustAccAddressFromBech32(delegation.DelegatorAddress)

	k.BeforeDelegationRemoved(ctx, delegatorAddress, delegation.GetValidatorAddr())
	_ = k.Delegations.Delete(ctx, collections.Join(delegatorAddress, delegation.GetValidatorAddr())) // idk if should panic or not
}

// GetUnbondingDelegation returns a unbonding delegation.
func (k Keeper) GetUnbondingDelegation(
	ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress,
) (ubd types.UnbondingDelegation, found bool) {
	ubd, err := k.UnbondingDelegations.Get(ctx, collections.Join(delAddr, valAddr))

	return ubd, err == nil
}

// GetUnbondingDelegationsFromValidator returns all unbonding delegations from a
// particular validator.
func (k Keeper) GetUnbondingDelegationsFromValidator(ctx sdk.Context, valAddr sdk.ValAddress) (ubds []types.UnbondingDelegation) {
	keys := k.UnbondingDelegations.Indexes.ValAddress.ExactMatch(ctx, valAddr).PrimaryKeys()
	ubds = make([]types.UnbondingDelegation, len(keys))
	for i, key := range keys {
		ubd, err := k.UnbondingDelegations.Get(ctx, key)
		if err != nil {
			panic(err)
		}
		ubds[i] = ubd
	}
	return
}

// HasMaxUnbondingDelegationEntries - check if unbonding delegation has maximum number of entries.
func (k Keeper) HasMaxUnbondingDelegationEntries(ctx sdk.Context,
	delegatorAddr sdk.AccAddress, validatorAddr sdk.ValAddress,
) bool {
	ubd, found := k.GetUnbondingDelegation(ctx, delegatorAddr, validatorAddr)
	if !found {
		return false
	}

	return len(ubd.Entries) >= int(k.MaxEntries(ctx))
}

// SetUnbondingDelegationEntry adds an entry to the unbonding delegation at
// the given addresses. It creates the unbonding delegation if it does not exist.
func (k Keeper) SetUnbondingDelegationEntry(
	ctx sdk.Context, delegatorAddr sdk.AccAddress, validatorAddr sdk.ValAddress,
	creationHeight int64, minTime time.Time, balance sdk.Int,
) types.UnbondingDelegation {
	ubd, found := k.GetUnbondingDelegation(ctx, delegatorAddr, validatorAddr)
	if found {
		ubd.AddEntry(creationHeight, minTime, balance)
	} else {
		ubd = types.NewUnbondingDelegation(delegatorAddr, validatorAddr, creationHeight, minTime, balance)
	}

	k.UnbondingDelegations.Insert(ctx, collections.Join(delegatorAddr, validatorAddr), ubd)

	return ubd
}

// unbonding delegation queue timeslice operations

// InsertUBDQueue inserts an unbonding delegation to the appropriate timeslice
// in the unbonding queue.
func (k Keeper) InsertUBDQueue(ctx sdk.Context, ubd types.UnbondingDelegation,
	completionTime time.Time,
) {
	dvPair := types.DVPair{DelegatorAddress: ubd.DelegatorAddress, ValidatorAddress: ubd.ValidatorAddress}

	k.UnbondingQueues.Insert(ctx, completionTime, types.DVPairs{Pairs: append(k.UnbondingQueues.GetOr(ctx, completionTime, types.DVPairs{}).Pairs, dvPair)})
}

// DequeueAllMatureUBDQueue returns a concatenated list of all the timeslices inclusively previous to
// currTime, and deletes the timeslices from the queue.
func (k Keeper) DequeueAllMatureUBDQueue(ctx sdk.Context, currTime time.Time) (matureUnbonds []types.DVPair) {
	for _, kv := range k.UnbondingQueues.Iterate(ctx, collections.Range[time.Time]{}.EndInclusive(ctx.BlockTime())).KeyValues() {
		k.UnbondingQueues.Delete(ctx, kv.Key)
		matureUnbonds = append(matureUnbonds, kv.Value.Pairs...)
	}
	return
}

// GetRedelegations returns a given amount of all the delegator redelegations.
func (k Keeper) GetRedelegations(ctx sdk.Context, delegator sdk.AccAddress,
	maxRetrieve uint16,
) (redelegations []types.Redelegation) {
	return k.Redelegations.Iterate(ctx, collections.TripletRange[sdk.AccAddress, sdk.ValAddress, sdk.ValAddress]{}.Prefix(delegator)).Values()
}

// GetRedelegation returns a redelegation.
func (k Keeper) GetRedelegation(ctx sdk.Context,
	delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress,
) (red types.Redelegation, found bool) {

	red, err := k.Redelegations.Get(ctx, collections.Join3(delAddr, valSrcAddr, valDstAddr))

	return red, err == nil
}

// GetRedelegationsFromSrcValidator returns all redelegations from a particular
// validator.
func (k Keeper) GetRedelegationsFromSrcValidator(ctx sdk.Context, valAddr sdk.ValAddress) (reds []types.Redelegation) {
	keys := k.Redelegations.Indexes.Src.ExactMatch(ctx, valAddr).PrimaryKeys()
	for _, key := range keys {
		reds = append(reds, k.Redelegations.GetOr(ctx, key, types.Redelegation{}))
	}
	return
}

// HasReceivingRedelegation checks if validator is receiving a redelegation.
func (k Keeper) HasReceivingRedelegation(ctx sdk.Context,
	delAddr sdk.AccAddress, valDstAddr sdk.ValAddress,
) bool {
	// redelegation{delegator: user1, src: val1, dst: val2}
	// TODO this is inefficient, we can do a smarter query with indexers.Dst.Iterate
	// but too lazy rn.
	// Triplet{delegator, source, destination}
	keys := k.Redelegations.Indexes.Dst.ExactMatch(ctx, valDstAddr).PrimaryKeys()
	for _, key := range keys {
		if key.K1().Equals(delAddr) {
			return true
		}
	}
	return false
}

// HasMaxRedelegationEntries checks if redelegation has maximum number of entries.
func (k Keeper) HasMaxRedelegationEntries(ctx sdk.Context,
	delegatorAddr sdk.AccAddress, validatorSrcAddr,
	validatorDstAddr sdk.ValAddress,
) bool {
	red, found := k.GetRedelegation(ctx, delegatorAddr, validatorSrcAddr, validatorDstAddr)
	if !found {
		return false
	}

	return len(red.Entries) >= int(k.MaxEntries(ctx))
}

// SetRedelegation set a redelegation and associated index.
func (k Keeper) SetRedelegation(ctx sdk.Context, red types.Redelegation) {
	delegatorAddress := sdk.MustAccAddressFromBech32(red.DelegatorAddress)
	valSrcAddr, err := sdk.ValAddressFromBech32(red.ValidatorSrcAddress)
	if err != nil {
		panic(err)
	}
	valDestAddr, err := sdk.ValAddressFromBech32(red.ValidatorDstAddress)
	if err != nil {
		panic(err)
	}

	k.Redelegations.Insert(ctx, collections.Join3(delegatorAddress, valSrcAddr, valDestAddr), red)

}

// SetRedelegationEntry adds an entry to the unbonding delegation at the given
// addresses. It creates the unbonding delegation if it does not exist.
func (k Keeper) SetRedelegationEntry(ctx sdk.Context,
	delegatorAddr sdk.AccAddress, validatorSrcAddr,
	validatorDstAddr sdk.ValAddress, creationHeight int64,
	minTime time.Time, balance sdk.Int,
	sharesSrc, sharesDst sdk.Dec,
) types.Redelegation {
	red, found := k.GetRedelegation(ctx, delegatorAddr, validatorSrcAddr, validatorDstAddr)
	if found {
		red.AddEntry(creationHeight, minTime, balance, sharesDst)
	} else {
		red = types.NewRedelegation(delegatorAddr, validatorSrcAddr,
			validatorDstAddr, creationHeight, minTime, balance, sharesDst)
	}

	k.SetRedelegation(ctx, red)

	return red
}

// RemoveRedelegation removes a redelegation object and associated index.
func (k Keeper) RemoveRedelegation(ctx sdk.Context, red types.Redelegation) {
	delegatorAddress := sdk.MustAccAddressFromBech32(red.DelegatorAddress)

	valSrcAddr, err := sdk.ValAddressFromBech32(red.ValidatorSrcAddress)
	if err != nil {
		panic(err)
	}
	valDestAddr, err := sdk.ValAddressFromBech32(red.ValidatorDstAddress)
	if err != nil {
		panic(err)
	}
	k.Redelegations.Delete(ctx, collections.Join3(delegatorAddress, valSrcAddr, valDestAddr))
}

// InsertRedelegationQueue insert an redelegation delegation to the appropriate
// timeslice in the redelegation queue.
func (k Keeper) InsertRedelegationQueue(ctx sdk.Context, red types.Redelegation,
	completionTime time.Time,
) {
	dvvTriplet := types.DVVTriplet{
		DelegatorAddress:    red.DelegatorAddress,
		ValidatorSrcAddress: red.ValidatorSrcAddress,
		ValidatorDstAddress: red.ValidatorDstAddress,
	}
	k.RedelegationQueues.Insert(
		ctx,
		completionTime,
		types.DVVTriplets{
			Triplets: append(k.RedelegationQueues.GetOr(ctx, completionTime, types.DVVTriplets{}).Triplets, dvvTriplet),
		},
	)
}

// DequeueAllMatureRedelegationQueue returns a concatenated list of all the
// timeslices inclusively previous to currTime, and deletes the timeslices from
// the queue.
func (k Keeper) DequeueAllMatureRedelegationQueue(ctx sdk.Context, currTime time.Time) (matureRedelegations []types.DVVTriplet) {
	for _, kv := range k.RedelegationQueues.Iterate(ctx, collections.Range[time.Time]{}.EndInclusive(currTime)).KeyValues() {
		k.RedelegationQueues.Delete(ctx, kv.Key)
		matureRedelegations = append(matureRedelegations, kv.Value.Triplets...)
	}
	return matureRedelegations
}

// Delegate performs a delegation, set/update everything necessary within the store.
// tokenSrc indicates the bond status of the incoming funds.
func (k Keeper) Delegate(
	ctx sdk.Context, delAddr sdk.AccAddress, bondAmt sdk.Int, tokenSrc types.BondStatus,
	validator types.Validator, subtractAccount bool,
) (newShares sdk.Dec, err error) {
	// In some situations, the exchange rate becomes invalid, e.g. if
	// Validator loses all tokens due to slashing. In this case,
	// make all future delegations invalid.
	if validator.InvalidExRate() {
		return sdk.ZeroDec(), types.ErrDelegatorShareExRateInvalid
	}

	// Get or create the delegation object
	valAddr := validator.GetOperator()
	delegation, err := k.Delegations.Get(ctx, collections.Join(delAddr, valAddr))
	if err != nil {
		delegation = types.NewDelegation(delAddr, validator.GetOperator(), sdk.ZeroDec())
	}

	// call the appropriate hook if present
	if err == nil {
		k.BeforeDelegationSharesModified(ctx, delAddr, validator.GetOperator())
	} else {
		k.BeforeDelegationCreated(ctx, delAddr, validator.GetOperator())
	}

	delegatorAddress := sdk.MustAccAddressFromBech32(delegation.DelegatorAddress)

	// if subtractAccount is true then we are
	// performing a delegation and not a redelegation, thus the source tokens are
	// all non bonded
	if subtractAccount {
		if tokenSrc == types.Bonded {
			panic("delegation token source cannot be bonded")
		}

		var sendName string

		switch {
		case validator.IsBonded():
			sendName = types.BondedPoolName
		case validator.IsUnbonding(), validator.IsUnbonded():
			sendName = types.NotBondedPoolName
		default:
			panic("invalid validator status")
		}

		coins := sdk.NewCoins(sdk.NewCoin(k.BondDenom(ctx), bondAmt))
		if err := k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, delegatorAddress, sendName, coins); err != nil {
			return sdk.Dec{}, err
		}
	} else {
		// potentially transfer tokens between pools, if
		switch {
		case tokenSrc == types.Bonded && validator.IsBonded():
			// do nothing
		case (tokenSrc == types.Unbonded || tokenSrc == types.Unbonding) && !validator.IsBonded():
			// do nothing
		case (tokenSrc == types.Unbonded || tokenSrc == types.Unbonding) && validator.IsBonded():
			// transfer pools
			k.notBondedTokensToBonded(ctx, bondAmt)
		case tokenSrc == types.Bonded && !validator.IsBonded():
			// transfer pools
			k.bondedTokensToNotBonded(ctx, bondAmt)
		default:
			panic("unknown token source bond status")
		}
	}

	_, newShares = k.AddValidatorTokensAndShares(ctx, validator, bondAmt)

	// Update delegation
	delegation.Shares = delegation.Shares.Add(newShares)
	k.Delegations.Insert(ctx, collections.Join(delAddr, valAddr), delegation)

	// Call the after-modification hook
	k.AfterDelegationModified(ctx, delegatorAddress, delegation.GetValidatorAddr())

	return newShares, nil
}

// Unbond unbonds a particular delegation and perform associated store operations.
func (k Keeper) Unbond(
	ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, shares sdk.Dec,
) (amount sdk.Int, err error) {
	// check if a delegation object exists in the store
	delegation, found := k.GetDelegation(ctx, delAddr, valAddr)
	if !found {
		return amount, types.ErrNoDelegatorForAddress
	}

	// call the before-delegation-modified hook
	k.BeforeDelegationSharesModified(ctx, delAddr, valAddr)

	// ensure that we have enough shares to remove
	if delegation.Shares.LT(shares) {
		return amount, sdkerrors.Wrap(types.ErrNotEnoughDelegationShares, delegation.Shares.String())
	}

	// get validator
	validator, found := k.GetValidator(ctx, valAddr)
	if !found {
		return amount, types.ErrNoValidatorFound
	}

	// subtract shares from delegation
	delegation.Shares = delegation.Shares.Sub(shares)

	delegatorAddress, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
	if err != nil {
		return amount, err
	}

	isValidatorOperator := delegatorAddress.Equals(validator.GetOperator())

	// If the delegation is the operator of the validator and undelegating will decrease the validator's
	// self-delegation below their minimum, we jail the validator.
	if isValidatorOperator && !validator.Jailed &&
		validator.TokensFromShares(delegation.Shares).TruncateInt().LT(validator.MinSelfDelegation) {
		k.jailValidator(ctx, validator)
		validator = k.mustGetValidator(ctx, validator.GetOperator())
	}

	// remove the delegation
	if delegation.Shares.IsZero() {
		k.RemoveDelegation(ctx, delegation)
	} else {
		k.Delegations.Insert(ctx, collections.Join(delAddr, valAddr), delegation)
		// call the after delegation modification hook
		k.AfterDelegationModified(ctx, delegatorAddress, delegation.GetValidatorAddr())
	}

	// remove the shares and coins from the validator
	// NOTE that the amount is later (in keeper.Delegation) moved between staking module pools
	validator, amount = k.RemoveValidatorTokensAndShares(ctx, validator, shares)

	if validator.DelegatorShares.IsZero() && validator.IsUnbonded() {
		// if not unbonded, we must instead remove validator in EndBlocker once it finishes its unbonding period
		k.RemoveValidator(ctx, validator.GetOperator())
	}

	return amount, nil
}

// getBeginInfo returns the completion time and height of a redelegation, along
// with a boolean signaling if the redelegation is complete based on the source
// validator.
func (k Keeper) getBeginInfo(
	ctx sdk.Context, valSrcAddr sdk.ValAddress,
) (completionTime time.Time, height int64, completeNow bool) {
	validator, found := k.GetValidator(ctx, valSrcAddr)

	// TODO: When would the validator not be found?
	switch {
	case !found || validator.IsBonded():
		// the longest wait - just unbonding period from now
		completionTime = ctx.BlockHeader().Time.Add(k.UnbondingTime(ctx))
		height = ctx.BlockHeight()

		return completionTime, height, false

	case validator.IsUnbonded():
		return completionTime, height, true

	case validator.IsUnbonding():
		return validator.UnbondingTime, validator.UnbondingHeight, false

	default:
		panic(fmt.Sprintf("unknown validator status: %s", validator.Status))
	}
}

// Undelegate unbonds an amount of delegator shares from a given validator. It
// will verify that the unbonding entries between the delegator and validator
// are not exceeded and unbond the staked tokens (based on shares) by creating
// an unbonding object and inserting it into the unbonding queue which will be
// processed during the staking EndBlocker.
func (k Keeper) Undelegate(
	ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount sdk.Dec,
) (time.Time, error) {
	validator, found := k.GetValidator(ctx, valAddr)
	if !found {
		return time.Time{}, types.ErrNoDelegatorForAddress
	}

	if k.HasMaxUnbondingDelegationEntries(ctx, delAddr, valAddr) {
		return time.Time{}, types.ErrMaxUnbondingDelegationEntries
	}

	returnAmount, err := k.Unbond(ctx, delAddr, valAddr, sharesAmount)
	if err != nil {
		return time.Time{}, err
	}

	// transfer the validator tokens to the not bonded pool
	if validator.IsBonded() {
		k.bondedTokensToNotBonded(ctx, returnAmount)
	}

	completionTime := ctx.BlockHeader().Time.Add(k.UnbondingTime(ctx))
	ubd := k.SetUnbondingDelegationEntry(ctx, delAddr, valAddr, ctx.BlockHeight(), completionTime, returnAmount)
	k.InsertUBDQueue(ctx, ubd, completionTime)

	return completionTime, nil
}

// CompleteUnbonding completes the unbonding of all mature entries in the
// retrieved unbonding delegation object and returns the total unbonding balance
// or an error upon failure.
func (k Keeper) CompleteUnbonding(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (sdk.Coins, error) {
	ubd, found := k.GetUnbondingDelegation(ctx, delAddr, valAddr)
	if !found {
		return nil, types.ErrNoUnbondingDelegation
	}

	bondDenom := k.GetParams(ctx).BondDenom
	balances := sdk.NewCoins()
	ctxTime := ctx.BlockHeader().Time

	delegatorAddress, err := sdk.AccAddressFromBech32(ubd.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	// loop through all the entries and complete unbonding mature entries
	for i := 0; i < len(ubd.Entries); i++ {
		entry := ubd.Entries[i]
		if entry.IsMature(ctxTime) {
			ubd.RemoveEntry(int64(i))
			i--

			// track undelegation only when remaining or truncated shares are non-zero
			if !entry.Balance.IsZero() {
				amt := sdk.NewCoin(bondDenom, entry.Balance)
				if err := k.bankKeeper.UndelegateCoinsFromModuleToAccount(
					ctx, types.NotBondedPoolName, delegatorAddress, sdk.NewCoins(amt),
				); err != nil {
					return nil, err
				}

				balances = balances.Add(amt)
			}
		}
	}

	// set the unbonding delegation or remove it if there are no more entries
	if len(ubd.Entries) == 0 {
		_ = k.UnbondingDelegations.Delete(ctx, collections.Join(delAddr, valAddr))
	} else {
		k.UnbondingDelegations.Insert(ctx, collections.Join(delAddr, valAddr), ubd)
	}

	return balances, nil
}

// BeginRedelegation begins unbonding / redelegation and creates a redelegation
// record.
func (k Keeper) BeginRedelegation(
	ctx sdk.Context, delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress, sharesAmount sdk.Dec,
) (completionTime time.Time, err error) {
	if bytes.Equal(valSrcAddr, valDstAddr) {
		return time.Time{}, types.ErrSelfRedelegation
	}

	dstValidator, found := k.GetValidator(ctx, valDstAddr)
	if !found {
		return time.Time{}, types.ErrBadRedelegationDst
	}

	srcValidator, found := k.GetValidator(ctx, valSrcAddr)
	if !found {
		return time.Time{}, types.ErrBadRedelegationDst
	}

	// check if this is a transitive redelegation
	if k.HasReceivingRedelegation(ctx, delAddr, valSrcAddr) {
		return time.Time{}, types.ErrTransitiveRedelegation
	}

	if k.HasMaxRedelegationEntries(ctx, delAddr, valSrcAddr, valDstAddr) {
		return time.Time{}, types.ErrMaxRedelegationEntries
	}

	returnAmount, err := k.Unbond(ctx, delAddr, valSrcAddr, sharesAmount)
	if err != nil {
		return time.Time{}, err
	}

	if returnAmount.IsZero() {
		return time.Time{}, types.ErrTinyRedelegationAmount
	}

	sharesCreated, err := k.Delegate(ctx, delAddr, returnAmount, srcValidator.GetStatus(), dstValidator, false)
	if err != nil {
		return time.Time{}, err
	}

	// create the unbonding delegation
	completionTime, height, completeNow := k.getBeginInfo(ctx, valSrcAddr)

	if completeNow { // no need to create the redelegation object
		return completionTime, nil
	}

	red := k.SetRedelegationEntry(
		ctx, delAddr, valSrcAddr, valDstAddr,
		height, completionTime, returnAmount, sharesAmount, sharesCreated,
	)
	k.InsertRedelegationQueue(ctx, red, completionTime)

	return completionTime, nil
}

// CompleteRedelegation completes the redelegations of all mature entries in the
// retrieved redelegation object and returns the total redelegation (initial)
// balance or an error upon failure.
func (k Keeper) CompleteRedelegation(
	ctx sdk.Context, delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress,
) (sdk.Coins, error) {
	red, found := k.GetRedelegation(ctx, delAddr, valSrcAddr, valDstAddr)
	if !found {
		return nil, types.ErrNoRedelegation
	}

	bondDenom := k.GetParams(ctx).BondDenom
	balances := sdk.NewCoins()
	ctxTime := ctx.BlockHeader().Time

	// loop through all the entries and complete mature redelegation entries
	for i := 0; i < len(red.Entries); i++ {
		entry := red.Entries[i]
		if entry.IsMature(ctxTime) {
			red.RemoveEntry(int64(i))
			i--

			if !entry.InitialBalance.IsZero() {
				balances = balances.Add(sdk.NewCoin(bondDenom, entry.InitialBalance))
			}
		}
	}

	// set the redelegation or remove it if there are no more entries
	if len(red.Entries) == 0 {
		k.RemoveRedelegation(ctx, red)
	} else {
		k.SetRedelegation(ctx, red)
	}

	return balances, nil
}

// ValidateUnbondAmount validates that a given unbond or redelegation amount is
// valied based on upon the converted shares. If the amount is valid, the total
// amount of respective shares is returned, otherwise an error is returned.
func (k Keeper) ValidateUnbondAmount(
	ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt sdk.Int,
) (shares sdk.Dec, err error) {
	validator, found := k.GetValidator(ctx, valAddr)
	if !found {
		return shares, types.ErrNoValidatorFound
	}

	del, found := k.GetDelegation(ctx, delAddr, valAddr)
	if !found {
		return shares, types.ErrNoDelegation
	}

	shares, err = validator.SharesFromTokens(amt)
	if err != nil {
		return shares, err
	}

	sharesTruncated, err := validator.SharesFromTokensTruncated(amt)
	if err != nil {
		return shares, err
	}

	delShares := del.GetShares()
	if sharesTruncated.GT(delShares) {
		return shares, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid shares amount")
	}

	// Cap the shares at the delegation's shares. Shares being greater could occur
	// due to rounding, however we don't want to truncate the shares or take the
	// minimum because we want to allow for the full withdraw of shares from a
	// delegation.
	if shares.GT(delShares) {
		shares = delShares
	}

	return shares, nil
}
