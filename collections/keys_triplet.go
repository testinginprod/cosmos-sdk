package collections

import "strings"

func Join3[K1, K2, K3 any](k1 K1, k2 K2, k3 K3) Triplet[K1, K2, K3] {
	return Triplet[K1, K2, K3]{
		k1: &k1,
		k2: &k2,
		k3: &k3,
	}
}

type Triplet[K1, K2, K3 any] struct {
	k1 *K1
	k2 *K2
	k3 *K3
}

func (k Triplet[K1, K2, K3]) K1() (k1 K1) {
	if k.k1 != nil {
		k1 = *k.k1
	}
	return
}

func (k Triplet[K1, K2, K3]) K2() (k2 K2) {
	if k.k2 != nil {
		k2 = *k.k2
	}
	return
}

func (k Triplet[K1, K2, K3]) K3() (k3 K3) {
	if k.k3 != nil {
		k3 = *k.k3
	}
	return
}

func TripletKeyEncoder[K1, K2, K3 any](k1c KeyEncoder[K1], k2c KeyEncoder[K2], k3c KeyEncoder[K3]) KeyEncoder[Triplet[K1, K2, K3]] {
	return tripletKeyEncoder[K1, K2, K3]{
		k1c: k1c,
		k2c: k2c,
		k3c: k3c,
	}
}

type tripletKeyEncoder[K1, K2, K3 any] struct {
	k1c KeyEncoder[K1]
	k2c KeyEncoder[K2]
	k3c KeyEncoder[K3]
}

func (t tripletKeyEncoder[K1, K2, K3]) Encode(key Triplet[K1, K2, K3]) []byte {
	if t.k1c != nil && t.k2c == nil && t.k3c != nil {
		panic("sub prefix missing from triplet")
	}

	var buf []byte
	if key.k1 != nil {
		buf = append(buf, t.k1c.Encode(*key.k1)...)
	}
	if key.k2 != nil {
		buf = append(buf, t.k2c.Encode(*key.k2)...)
	}
	if key.k3 != nil {
		buf = append(buf, t.k3c.Encode(*key.k3)...)
	}
	return buf
}

func (t tripletKeyEncoder[K1, K2, K3]) Decode(b []byte) (int, Triplet[K1, K2, K3]) {
	i1, k1 := t.k1c.Decode(b)
	i2, k2 := t.k2c.Decode(b[i1:])
	i3, k3 := t.k3c.Decode(b[i1+i2:])
	return i1 + i2 + i3, Triplet[K1, K2, K3]{
		k1: &k1,
		k2: &k2,
		k3: &k3,
	}
}

func (t tripletKeyEncoder[K1, K2, K3]) Stringify(key Triplet[K1, K2, K3]) string {
	s := strings.Builder{}
	s.WriteByte('(')
	if key.k1 == nil {
		s.WriteString("<nil>")
	} else {
		s.WriteByte('"')
		s.WriteString(t.k1c.Stringify(*key.k1))
		s.WriteByte('"')
	}
	s.WriteByte(',')
	s.WriteByte(' ')
	if key.k2 == nil {
		s.WriteString("<nil>")
	} else {
		s.WriteByte('"')
		s.WriteString(t.k2c.Stringify(*key.k2))
		s.WriteByte('"')
	}
	s.WriteByte(',')
	s.WriteByte(' ')
	if key.k3 == nil {
		s.WriteString("<nil>")
	} else {
		s.WriteByte('"')
		s.WriteString(t.k3c.Stringify(*key.k3))
		s.WriteByte('"')
	}
	s.WriteByte(')')
	return s.String()
}

type TripletRange[K1, K2, K3 any] struct {
	prefix    *K1
	subPrefix *K2
	start     *Bound[K3]
	end       *Bound[K3]
	order     Order
}

// Prefix makes the range contain only keys starting with the given k1 prefix.
func (p TripletRange[K1, K2, K3]) Prefix(prefix K1) TripletRange[K1, K2, K3] {
	p.prefix = &prefix
	return p
}

func (p TripletRange[K1, K2, K3]) SubPrefix(subPrefix K2) TripletRange[K1, K2, K3] {
	p.subPrefix = &subPrefix
	return p
}

// StartInclusive makes the range contain only keys which are bigger or equal to the provided start K3.
func (p TripletRange[K1, K2, K3]) StartInclusive(start K3) TripletRange[K1, K2, K3] {
	p.start = BoundInclusive(start)
	return p
}

// StartExclusive makes the range contain only keys which are bigger to the provided start K3.
func (p TripletRange[K1, K2, K3]) StartExclusive(start K3) TripletRange[K1, K2, K3] {
	p.start = BoundExclusive(start)
	return p
}

// EndInclusive makes the range contain only keys which are smaller or equal to the provided end K3.
func (p TripletRange[K1, K2, K3]) EndInclusive(end K3) TripletRange[K1, K2, K3] {
	p.end = BoundInclusive(end)
	return p
}

// EndExclusive makes the range contain only keys which are smaller to the provided end K3.
func (p TripletRange[K1, K2, K3]) EndExclusive(end K3) TripletRange[K1, K2, K3] {
	p.end = BoundExclusive(end)
	return p
}

// Descending makes the range run in reverse (bigger->smaller, instead of smaller->bigger)
func (p TripletRange[K1, K2, K3]) Descending() TripletRange[K1, K2, K3] {
	p.order = OrderDescending
	return p
}

// RangeValues implements Ranger for Triplet[K1, K2, K3].
// If start and end are set, prefix and sub-prefix must be set too or the function call will panic.
// If sub-prefix is set then prefix must be set too.
// The implementation returns a range which prefixes over the K1 prefix and K2 sub-prefix (if defined).
// And the key range goes from K3 start to K3 end (if any are defined).
func (p TripletRange[K1, K2, K3]) RangeValues() (prefix *Triplet[K1, K2, K3], start *Bound[Triplet[K1, K2, K3]], end *Bound[Triplet[K1, K2, K3]], order Order) {
	if p.prefix == nil && p.subPrefix != nil {
		panic("invalid TripletRange usage: if sub prefix is set then prefix must be set too")
	}

	if (p.end != nil || p.start != nil) && p.subPrefix == nil {
		panic("invalid PairRange usage: if end or start are set, prefix must be set too")
	}
	if p.prefix != nil {
		prefix = new(Triplet[K1, K2, K3])
		prefix.k1 = p.prefix
	}
	if p.subPrefix != nil {
		prefix.k2 = p.subPrefix
	}
	if p.start != nil {
		start = &Bound[Triplet[K1, K2, K3]]{
			value:     Triplet[K1, K2, K3]{k3: &p.start.value},
			inclusive: p.start.inclusive,
		}
	}
	if p.end != nil {
		end = &Bound[Triplet[K1, K2, K3]]{
			value:     Triplet[K1, K2, K3]{k3: &p.end.value},
			inclusive: p.end.inclusive,
		}
	}

	order = p.order
	return
}
