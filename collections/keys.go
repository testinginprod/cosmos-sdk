package collections

import (
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	// StringKeyEncoder can be used to encode string keys.
	StringKeyEncoder KeyEncoder[string] = stringKey{}
	// AccAddressKeyEncoder can be used to encode sdk.AccAddress keys.
	AccAddressKeyEncoder KeyEncoder[sdk.AccAddress] = accAddressKey{}
	// ValAddressKeyEncoder can be used to encode sdk.ValAddress keys.
	ValAddressKeyEncoder KeyEncoder[sdk.ValAddress] = valAddressKey{}
	// ConsAddressKeyEncoder
	ConsAddressKeyEncoder KeyEncoder[sdk.ConsAddress] = consAddressKey{}
	// TimeKeyEncoder can be used to encode time.Time keys.
	TimeKeyEncoder KeyEncoder[time.Time] = timeKey{}
	// Uint64KeyEncoder can be used to encode uint64 keys.
	Uint64KeyEncoder KeyEncoder[uint64] = uint64Key{}
)

type stringKey struct{}

func (stringKey) Encode(s string) []byte {
	if err := validString(s); err != nil {
		panic(fmt.Errorf("invalid StringKey: %w", err))
	}
	return append([]byte(s), 0) // null terminate it for safe prefixing
}

func (stringKey) Decode(b []byte) (int, string) {
	l := len(b)
	if l < 2 {
		panic("invalid StringKey bytes")
	}
	for i, c := range b {
		if c == 0 {
			return i + 1, string(b[:i])
		}
	}
	panic(fmt.Errorf("string is not null terminated: %s", b))
}

type uint64Key struct{}

func (uint64Key) Stringify(u uint64) string     { return strconv.FormatUint(u, 10) }
func (uint64Key) Encode(u uint64) []byte        { return sdk.Uint64ToBigEndian(u) }
func (uint64Key) Decode(b []byte) (int, uint64) { return 8, sdk.BigEndianToUint64(b) }

type timeKey struct{}

func (timeKey) Stringify(t time.Time) string { return t.String() }
func (timeKey) Encode(t time.Time) []byte    { return sdk.FormatTimeBytes(t) }
func (timeKey) Decode(b []byte) (int, time.Time) {
	t, err := sdk.ParseTimeBytes(b)
	if err != nil {
		panic(err)
	}
	return len(b), t
}

type accAddressKey struct{}

func (accAddressKey) Stringify(addr sdk.AccAddress) string { return addr.String() }
func (accAddressKey) Encode(addr sdk.AccAddress) []byte {
	return StringKeyEncoder.Encode(addr.String())
}
func (accAddressKey) Decode(b []byte) (int, sdk.AccAddress) {
	i, s := StringKeyEncoder.Decode(b)
	return i, sdk.MustAccAddressFromBech32(s)
}

type valAddressKey struct{}

func (v valAddressKey) Encode(key sdk.ValAddress) []byte {
	return StringKeyEncoder.Encode(key.String())
}

func (v valAddressKey) Decode(b []byte) (int, sdk.ValAddress) {
	r, k := StringKeyEncoder.Decode(b)
	val, err := sdk.ValAddressFromBech32(k)
	if err != nil {
		panic(err)
	}
	return r, val
}

type consAddressKey struct{}

func (consAddressKey) Encode(key sdk.ConsAddress) []byte {
	return StringKeyEncoder.Encode(key.String())
}

func (consAddressKey) Decode(b []byte) (int, sdk.ConsAddress) {
	r, k := StringKeyEncoder.Decode(b)
	cons, err := sdk.ConsAddressFromBech32(k)
	if err != nil {
		panic(err)
	}
	return r, cons
}

func (consAddressKey) Stringify(key sdk.ConsAddress) string { return key.String() }

func (v valAddressKey) Stringify(key sdk.ValAddress) string {
	return key.String()
}

func (stringKey) Stringify(s string) string {
	return s
}

func validString(s string) error {
	for i, c := range s {
		if c == 0 {
			return fmt.Errorf("invalid null character at index %d: %s", i, s)
		}
	}
	return nil
}
