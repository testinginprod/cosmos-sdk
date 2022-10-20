package types

import (
	"bytes"
	"fmt"
	"github.com/cosmos/cosmos-sdk/collections"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/cosmos/cosmos-sdk/types/kv"
)

const (
	// ModuleName is the name of the staking module
	ModuleName = "staking"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// QuerierRoute is the querier route for the staking module
	QuerierRoute = ModuleName

	// RouterKey is the msg router key for the staking module
	RouterKey = ModuleName
)

var (
	// Keys for store prefixes
	// Last* values are constant during a block.
	LastValidatorPowerKey = []byte{0x11} // prefix for each key to a validator index, for bonded validators

	ValidatorsKey             = collections.Namespace(0x21) // prefix for each key to a validator
	ValidatorsByConsAddrKey   = collections.Namespace(0x22) // prefix for each key to a validator index, by pubkey
	ValidatorsByPowerIndexKey = collections.Namespace(0x23) // prefix for each key to a validator index, sorted by power

	DelegationKey                    = collections.Namespace(0x31) // key for a delegation
	UnbondingDelegationKey           = collections.Namespace(0x32) // key for an unbonding-delegation
	UnbondingDelegationByValIndexKey = collections.Namespace(0x33) // prefix for each key for an unbonding-delegation, by validator operator
	RedelegationKey                  = collections.Namespace(0x34) // key for a redelegation
	RedelegationByValSrcIndexKey     = collections.Namespace(0x35) // prefix for each key for an redelegation, by source validator operator
	RedelegationByValDstIndexKey     = collections.Namespace(0x36) // prefix for each key for an redelegation, by destination validator operator

	UnbondingQueueKey    = collections.Namespace(0x41) // prefix for the timestamps in unbonding queue
	RedelegationQueueKey = collections.Namespace(0x42) // prefix for the timestamps in redelegations queue
	ValidatorQueueKey    = []byte{0x43}                // prefix for the timestamps in validator queue

	HistoricalInfoKey = collections.Namespace(0x50) // prefix for the historical info
)

// AddressFromValidatorsKey creates the validator operator address from ValidatorsKey
func AddressFromValidatorsKey(key []byte) []byte {
	kv.AssertKeyAtLeastLength(key, 3)
	return key[2:] // remove prefix bytes and address length
}

// AddressFromLastValidatorPowerKey creates the validator operator address from LastValidatorPowerKey
func AddressFromLastValidatorPowerKey(key []byte) []byte {
	kv.AssertKeyAtLeastLength(key, 3)
	return key[2:] // remove prefix bytes and address length
}

// GetLastValidatorPowerKey creates the bonded validator index key for an operator address
func GetLastValidatorPowerKey(operator sdk.ValAddress) []byte {
	return append(LastValidatorPowerKey, address.MustLengthPrefix(operator)...)
}

// GetValidatorQueueKey returns the prefix key used for getting a set of unbonding
// validators whose unbonding completion occurs at the given time and height.
func GetValidatorQueueKey(timestamp time.Time, height int64) []byte {
	heightBz := sdk.Uint64ToBigEndian(uint64(height))
	timeBz := sdk.FormatTimeBytes(timestamp)
	timeBzL := len(timeBz)
	prefixL := len(ValidatorQueueKey)

	bz := make([]byte, prefixL+8+timeBzL+8)

	// copy the prefix
	copy(bz[:prefixL], ValidatorQueueKey)

	// copy the encoded time bytes length
	copy(bz[prefixL:prefixL+8], sdk.Uint64ToBigEndian(uint64(timeBzL)))

	// copy the encoded time bytes
	copy(bz[prefixL+8:prefixL+8+timeBzL], timeBz)

	// copy the encoded height
	copy(bz[prefixL+8+timeBzL:], heightBz)

	return bz
}

// ParseValidatorQueueKey returns the encoded time and height from a key created
// from GetValidatorQueueKey.
func ParseValidatorQueueKey(bz []byte) (time.Time, int64, error) {
	prefixL := len(ValidatorQueueKey)
	if prefix := bz[:prefixL]; !bytes.Equal(prefix, ValidatorQueueKey) {
		return time.Time{}, 0, fmt.Errorf("invalid prefix; expected: %X, got: %X", ValidatorQueueKey, prefix)
	}

	timeBzL := sdk.BigEndianToUint64(bz[prefixL : prefixL+8])
	ts, err := sdk.ParseTimeBytes(bz[prefixL+8 : prefixL+8+int(timeBzL)])
	if err != nil {
		return time.Time{}, 0, err
	}

	height := sdk.BigEndianToUint64(bz[prefixL+8+int(timeBzL):])

	return ts, int64(height), nil
}
