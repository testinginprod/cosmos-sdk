package keeper

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/collections"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Implements ValidatorSet interface
var _ types.ValidatorSet = Keeper{}

// Implements DelegationSet interface
var _ types.DelegationSet = Keeper{}

type ValidatorsIndexes struct {
	ConsAddress collections.MultiIndex[sdk.ConsAddress, sdk.ValAddress, types.Validator]
	Power       collections.MultiIndex[int64, sdk.ValAddress, types.Validator]
}

func (v ValidatorsIndexes) IndexerList() []collections.Indexer[sdk.ValAddress, types.Validator] {
	return []collections.Indexer[sdk.ValAddress, types.Validator]{v.ConsAddress, v.Power}
}

type UnbondingDelegationsIndexes struct {
	ValAddress collections.MultiIndex[sdk.ValAddress, collections.Pair[sdk.AccAddress, sdk.ValAddress], types.UnbondingDelegation]
}

func (u UnbondingDelegationsIndexes) IndexerList() []collections.Indexer[collections.Pair[sdk.AccAddress, sdk.ValAddress], types.UnbondingDelegation] {
	return []collections.Indexer[collections.Pair[sdk.AccAddress, sdk.ValAddress], types.UnbondingDelegation]{u.ValAddress}
}

// keeper of the staking store
type Keeper struct {
	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	authKeeper types.AccountKeeper
	bankKeeper types.BankKeeper
	hooks      types.StakingHooks
	paramstore paramtypes.Subspace

	LastTotalPower       collections.Item[sdk.Int]
	HistoricalInfo       collections.Map[int64, types.HistoricalInfo]
	Validators           collections.IndexedMap[sdk.ValAddress, types.Validator, ValidatorsIndexes]
	Delegations          collections.Map[collections.Pair[sdk.AccAddress, sdk.ValAddress], types.Delegation]
	UnbondingDelegations collections.IndexedMap[collections.Pair[sdk.AccAddress, sdk.ValAddress], types.UnbondingDelegation, UnbondingDelegationsIndexes]
}

// NewKeeper creates a new staking Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, storeKey sdk.StoreKey, ak types.AccountKeeper, bk types.BankKeeper,
	ps paramtypes.Subspace,
) Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	// ensure bonded and not bonded module accounts are set
	if addr := ak.GetModuleAddress(types.BondedPoolName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.BondedPoolName))
	}

	if addr := ak.GetModuleAddress(types.NotBondedPoolName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.NotBondedPoolName))
	}

	return Keeper{
		storeKey:       storeKey,
		cdc:            cdc,
		authKeeper:     ak,
		bankKeeper:     bk,
		hooks:          nil,
		paramstore:     ps,
		LastTotalPower: collections.NewItem(storeKey, 12, collections.SDKIntValueEncoder),
		HistoricalInfo: collections.NewMap(storeKey, types.HistoricalInfoKey, collections.Int64KeyEncoder, collections.ProtoValueEncoder[types.HistoricalInfo](cdc)),
		Validators: collections.NewIndexedMap(storeKey, types.ValidatorsKey,
			collections.ValAddressKeyEncoder, collections.ProtoValueEncoder[types.Validator](cdc),
			ValidatorsIndexes{
				ConsAddress: collections.NewMultiIndex(
					storeKey, types.ValidatorsByConsAddrKey,
					collections.ConsAddressKeyEncoder, collections.ValAddressKeyEncoder,
					func(v types.Validator) sdk.ConsAddress {
						cAddr, err := v.GetConsAddr()
						if err != nil {
							panic(err)
						}
						return cAddr
					},
				),
				Power: collections.NewMultiIndex(
					storeKey, types.ValidatorsByPowerIndexKey,
					collections.Int64KeyEncoder, collections.ValAddressKeyEncoder,
					func(v types.Validator) int64 {
						return sdk.TokensToConsensusPower(v.Tokens, sdk.DefaultPowerReduction)
					},
				),
			},
		),
		Delegations: collections.NewMap(
			storeKey, types.DelegationKey,
			collections.PairKeyEncoder(collections.AccAddressKeyEncoder, collections.ValAddressKeyEncoder),
			collections.ProtoValueEncoder[types.Delegation](cdc),
		),
		UnbondingDelegations: collections.NewIndexedMap(
			storeKey, types.UnbondingDelegationKey,
			collections.PairKeyEncoder(collections.AccAddressKeyEncoder, collections.ValAddressKeyEncoder),
			collections.ProtoValueEncoder[types.UnbondingDelegation](cdc),
			UnbondingDelegationsIndexes{
				ValAddress: collections.NewMultiIndex(
					storeKey, types.UnbondingDelegationByValIndexKey,
					collections.ValAddressKeyEncoder,
					collections.PairKeyEncoder(collections.AccAddressKeyEncoder, collections.ValAddressKeyEncoder),
					func(v types.UnbondingDelegation) sdk.ValAddress {
						valAddr, err := sdk.ValAddressFromBech32(v.ValidatorAddress)
						if err != nil {
							panic(err)
						}
						return valAddr
					},
				),
			},
		),
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// Set the validator hooks
func (k *Keeper) SetHooks(sh types.StakingHooks) *Keeper {
	if k.hooks != nil {
		panic("cannot set validator hooks twice")
	}

	k.hooks = sh

	return k
}

// Load the last total validator power.
func (k Keeper) GetLastTotalPower(ctx sdk.Context) sdk.Int {
	return k.LastTotalPower.GetOr(ctx, sdk.ZeroInt())
}
