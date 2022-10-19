package collections

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
	"strconv"
)

type int64Value struct{}

func (i int64Value) Encode(value int64) []byte { return Uint64KeyEncoder.Encode(uint64(value)) }

func (i int64Value) Decode(b []byte) int64 {
	_, v := Uint64KeyEncoder.Decode(b)
	return int64(v)
}

func (i int64Value) Stringify(value int64) string { return strconv.FormatInt(value, 10) }

func (i int64Value) Name() string {
	//TODO implement me
	panic("implement me")
}

var (
	Int64ValueEncoder  ValueEncoder[int64]   = int64Value{}
	SDKIntValueEncoder ValueEncoder[sdk.Int] = sdkIntValueEncoder{}
)

// ProtoValueEncoder returns a protobuf value encoder given the codec.BinaryCodec.
// It's used to convert a specific protobuf object into bytes representation and convert
// the protobuf object bytes representation into the concrete object.
func ProtoValueEncoder[V any, PV interface {
	*V
	codec.ProtoMarshaler
}](cdc codec.BinaryCodec) ValueEncoder[V] {
	return protoValueEncoder[V, PV]{
		cdc: cdc,
	}
}

type protoValueEncoder[V any, PV interface {
	*V
	codec.ProtoMarshaler
}] struct {
	cdc codec.BinaryCodec
}

func (p protoValueEncoder[V, PV]) Name() string          { return proto.MessageName(PV(new(V))) }
func (p protoValueEncoder[V, PV]) Encode(value V) []byte { return p.cdc.MustMarshal(PV(&value)) }
func (p protoValueEncoder[V, PV]) Stringify(v V) string  { return PV(&v).String() }
func (p protoValueEncoder[V, PV]) Decode(b []byte) V {
	v := PV(new(V))
	p.cdc.MustUnmarshal(b, v)
	return *v
}

type sdkIntValueEncoder struct{}

func (s sdkIntValueEncoder) Encode(value sdk.Int) []byte {
	b, err := value.Marshal()
	if err != nil {
		panic(err)
	}
	return b
}

func (s sdkIntValueEncoder) Decode(b []byte) sdk.Int {
	i := new(sdk.Int)
	err := i.Unmarshal(b)
	if err != nil {
		panic(err)
	}
	return *i
}

func (s sdkIntValueEncoder) Stringify(value sdk.Int) string {
	return value.String()
}

func (s sdkIntValueEncoder) Name() string {
	return "sdk.Int"
}
