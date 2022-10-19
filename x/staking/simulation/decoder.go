package simulation

import (
	"bytes"
	"fmt"
	"github.com/cosmos/cosmos-sdk/collections"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/kv"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// NewDecodeStore returns a decoder function closure that unmarshals the KVPair's
// Value to the corresponding staking type.
func NewDecodeStore(cdc codec.Codec) func(kvA, kvB kv.Pair) string {
	return func(kvA, kvB kv.Pair) string {
		switch {
		case bytes.Equal(kvA.Key[:1], []byte{0x12}):
			return fmt.Sprintf("%v\n%v", collections.SDKIntValueEncoder.Decode(kvA.Value), collections.SDKIntValueEncoder.Decode(kvB.Value))
		case bytes.Equal(kvA.Key[:1], types.ValidatorsKey.Prefix()):
			var validatorA, validatorB types.Validator

			cdc.MustUnmarshal(kvA.Value, &validatorA)
			cdc.MustUnmarshal(kvB.Value, &validatorB)

			return fmt.Sprintf("%v\n%v", validatorA, validatorB)
		case bytes.Equal(kvA.Key[:1], types.LastValidatorPowerKey),
			bytes.Equal(kvA.Key[:1], types.ValidatorsByConsAddrKey.Prefix()),
			bytes.Equal(kvA.Key[:1], types.ValidatorsByPowerIndexKey):
			return fmt.Sprintf("%v\n%v", sdk.ValAddress(kvA.Value), sdk.ValAddress(kvB.Value))

		case bytes.Equal(kvA.Key[:1], types.DelegationKey.Prefix()):
			var delegationA, delegationB types.Delegation

			cdc.MustUnmarshal(kvA.Value, &delegationA)
			cdc.MustUnmarshal(kvB.Value, &delegationB)

			return fmt.Sprintf("%v\n%v", delegationA, delegationB)
		case bytes.Equal(kvA.Key[:1], types.UnbondingDelegationKey.Prefix()),
			bytes.Equal(kvA.Key[:1], types.UnbondingDelegationByValIndexKey.Prefix()):
			var ubdA, ubdB types.UnbondingDelegation

			cdc.MustUnmarshal(kvA.Value, &ubdA)
			cdc.MustUnmarshal(kvB.Value, &ubdB)

			return fmt.Sprintf("%v\n%v", ubdA, ubdB)
		case bytes.Equal(kvA.Key[:1], types.RedelegationKey),
			bytes.Equal(kvA.Key[:1], types.RedelegationByValSrcIndexKey):
			var redA, redB types.Redelegation

			cdc.MustUnmarshal(kvA.Value, &redA)
			cdc.MustUnmarshal(kvB.Value, &redB)

			return fmt.Sprintf("%v\n%v", redA, redB)
		default:
			panic(fmt.Sprintf("invalid staking key prefix %X", kvA.Key[:1]))
		}
	}
}
