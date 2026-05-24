package swisstable

// This file defines the per-group storage layout and the portable SWAR
// match operations on the packed control word. The same algebra Go's own
// runtime uses on non-AMD64 paths (see internal/runtime/maps/group.go).

const (
	groupSize = 8

	// Per-slot control-byte encoding.
	ctrlEmpty   uint8 = 0x80 // 1000_0000
	ctrlDeleted uint8 = 0xfe // 1111_1110
	// Full slot: 0xxx_xxxx — low 7 bits are H2 of the hash.

	bitsetLSB uint64 = 0x0101010101010101
	bitsetMSB uint64 = 0x8080808080808080

	// emptyCtrl is ctrlEmpty broadcast across all 8 bytes of a ctrl word.
	emptyCtrl = uint64(ctrlEmpty) * bitsetLSB
)

type slot[K comparable, V any] struct {
	key K
	val V
}

type group[K comparable, V any] struct {
	ctrl  uint64
	slots [groupSize]slot[K, V]
}

// Each match* function returns a uint64 bitmask: byte i has its high bit
// set iff slot i matches. Iterate matches via bits.TrailingZeros64 + &-1.

// matchH2 may emit false positives when h2==0; the subsequent key compare filters them out.
func matchH2(ctrl uint64, h2 uint8) uint64 {
	v := ctrl ^ (bitsetLSB * uint64(h2))
	return (v - bitsetLSB) &^ v & bitsetMSB
}

// matchEmpty: bit7 set, bit1 unset (distinguishes Empty 0x80 from Deleted 0xfe).
func matchEmpty(ctrl uint64) uint64 { return ctrl &^ (ctrl << 6) & bitsetMSB }

func matchEmptyOrDeleted(ctrl uint64) uint64 { return ctrl & bitsetMSB }

func matchFull(ctrl uint64) uint64 { return ^ctrl & bitsetMSB }

// getCtrl extracts the i-th control byte; the uint64->uint8 conversion is the truncation we want.
func getCtrl(ctrl uint64, i int) uint8 { return uint8(ctrl >> (i * 8)) } // #nosec G115 -- intentional byte truncation

func setCtrl(ctrl *uint64, i int, c uint8) {
	shift := uint(i * 8)
	*ctrl = (*ctrl &^ (uint64(0xff) << shift)) | (uint64(c) << shift)
}
