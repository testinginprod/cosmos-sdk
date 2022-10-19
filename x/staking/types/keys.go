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
	RedelegationKey                  = []byte{0x34}                // key for a redelegation
	RedelegationByValSrcIndexKey     = []byte{0x35}                // prefix for each key for an redelegation, by source validator operator
	RedelegationByValDstIndexKey     = []byte{0x36}                // prefix for each key for an redelegation, by destination validator operator

	UnbondingQueueKey    = []byte{0x41} // prefix for the timestamps in unbonding queue
	RedelegationQueueKey = []byte{0x42} // prefix for the timestamps in redelegations queue
	ValidatorQueueKey    = []byte{0x43} // prefix for the timestamps in validator queue

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

// GetUnbondingDelegationTimeKey creates the prefix for all unbonding delegations from a delegator
func GetUnbondingDelegationTimeKey(timestamp time.Time) []byte {
	bz := sdk.FormatTimeBytes(timestamp)
	return append(UnbondingQueueKey, bz...)
}

// GetREDKey returns a key prefix for indexing a redelegation from a delegator
// and source validator to a destination validator.
func GetREDKey(delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress) []byte {
	// key is of the form GetREDsKey || valSrcAddrLen (1 byte) || valSrcAddr || valDstAddrLen (1 byte) || valDstAddr
	key := make([]byte, 1+3+len(delAddr)+len(valSrcAddr)+len(valDstAddr))

	copy(key[0:2+len(delAddr)], GetREDsKey(delAddr.Bytes()))
	key[2+len(delAddr)] = byte(len(valSrcAddr))
	copy(key[3+len(delAddr):3+len(delAddr)+len(valSrcAddr)], valSrcAddr.Bytes())
	key[3+len(delAddr)+len(valSrcAddr)] = byte(len(valDstAddr))
	copy(key[4+len(delAddr)+len(valSrcAddr):], valDstAddr.Bytes())

	return key
}

// GetREDByValSrcIndexKey creates the index-key for a redelegation, stored by source-validator-index
// VALUE: none (key rearrangement used)
func GetREDByValSrcIndexKey(delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress) []byte {
	REDSFromValsSrcKey := GetREDsFromValSrcIndexKey(valSrcAddr)
	offset := len(REDSFromValsSrcKey)

	// key is of the form REDSFromValsSrcKey || delAddrLen (1 byte) || delAddr || valDstAddrLen (1 byte) || valDstAddr
	key := make([]byte, offset+2+len(delAddr)+len(valDstAddr))
	copy(key[0:offset], REDSFromValsSrcKey)
	key[offset] = byte(len(delAddr))
	copy(key[offset+1:offset+1+len(delAddr)], delAddr.Bytes())
	key[offset+1+len(delAddr)] = byte(len(valDstAddr))
	copy(key[offset+2+len(delAddr):], valDstAddr.Bytes())

	return key
}

// GetREDByValDstIndexKey creates the index-key for a redelegation, stored by destination-validator-index
// VALUE: none (key rearrangement used)
func GetREDByValDstIndexKey(delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress) []byte {
	REDSToValsDstKey := GetREDsToValDstIndexKey(valDstAddr)
	offset := len(REDSToValsDstKey)

	// key is of the form REDSToValsDstKey || delAddrLen (1 byte) || delAddr || valSrcAddrLen (1 byte) || valSrcAddr
	key := make([]byte, offset+2+len(delAddr)+len(valSrcAddr))
	copy(key[0:offset], REDSToValsDstKey)
	key[offset] = byte(len(delAddr))
	copy(key[offset+1:offset+1+len(delAddr)], delAddr.Bytes())
	key[offset+1+len(delAddr)] = byte(len(valSrcAddr))
	copy(key[offset+2+len(delAddr):], valSrcAddr.Bytes())

	return key
}

// GetREDKeyFromValSrcIndexKey rearranges the ValSrcIndexKey to get the REDKey
func GetREDKeyFromValSrcIndexKey(indexKey []byte) []byte {
	// note that first byte is prefix byte, which we remove
	kv.AssertKeyAtLeastLength(indexKey, 2)
	addrs := indexKey[1:]

	valSrcAddrLen := addrs[0]
	kv.AssertKeyAtLeastLength(addrs, int(valSrcAddrLen)+2)
	valSrcAddr := addrs[1 : valSrcAddrLen+1]
	delAddrLen := addrs[valSrcAddrLen+1]
	kv.AssertKeyAtLeastLength(addrs, int(valSrcAddrLen)+int(delAddrLen)+2)
	delAddr := addrs[valSrcAddrLen+2 : valSrcAddrLen+2+delAddrLen]
	kv.AssertKeyAtLeastLength(addrs, int(valSrcAddrLen)+int(delAddrLen)+4)
	valDstAddr := addrs[valSrcAddrLen+delAddrLen+3:]

	return GetREDKey(delAddr, valSrcAddr, valDstAddr)
}

// GetREDKeyFromValDstIndexKey rearranges the ValDstIndexKey to get the REDKey
func GetREDKeyFromValDstIndexKey(indexKey []byte) []byte {
	// note that first byte is prefix byte, which we remove
	kv.AssertKeyAtLeastLength(indexKey, 2)
	addrs := indexKey[1:]

	valDstAddrLen := addrs[0]
	kv.AssertKeyAtLeastLength(addrs, int(valDstAddrLen)+2)
	valDstAddr := addrs[1 : valDstAddrLen+1]
	delAddrLen := addrs[valDstAddrLen+1]
	kv.AssertKeyAtLeastLength(addrs, int(valDstAddrLen)+int(delAddrLen)+3)
	delAddr := addrs[valDstAddrLen+2 : valDstAddrLen+2+delAddrLen]
	kv.AssertKeyAtLeastLength(addrs, int(valDstAddrLen)+int(delAddrLen)+4)
	valSrcAddr := addrs[valDstAddrLen+delAddrLen+3:]

	return GetREDKey(delAddr, valSrcAddr, valDstAddr)
}

// GetRedelegationTimeKey returns a key prefix for indexing an unbonding
// redelegation based on a completion time.
func GetRedelegationTimeKey(timestamp time.Time) []byte {
	bz := sdk.FormatTimeBytes(timestamp)
	return append(RedelegationQueueKey, bz...)
}

// GetREDsKey returns a key prefix for indexing a redelegation from a delegator
// address.
func GetREDsKey(delAddr sdk.AccAddress) []byte {
	return append(RedelegationKey, address.MustLengthPrefix(delAddr)...)
}

// GetREDsFromValSrcIndexKey returns a key prefix for indexing a redelegation to
// a source validator.
func GetREDsFromValSrcIndexKey(valSrcAddr sdk.ValAddress) []byte {
	return append(RedelegationByValSrcIndexKey, address.MustLengthPrefix(valSrcAddr)...)
}

// GetREDsToValDstIndexKey returns a key prefix for indexing a redelegation to a
// destination (target) validator.
func GetREDsToValDstIndexKey(valDstAddr sdk.ValAddress) []byte {
	return append(RedelegationByValDstIndexKey, address.MustLengthPrefix(valDstAddr)...)
}

// GetREDsByDelToValDstIndexKey returns a key prefix for indexing a redelegation
// from an address to a source validator.
func GetREDsByDelToValDstIndexKey(delAddr sdk.AccAddress, valDstAddr sdk.ValAddress) []byte {
	return append(GetREDsToValDstIndexKey(valDstAddr), address.MustLengthPrefix(delAddr)...)
}
