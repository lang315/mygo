// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run gen.go

// Package psl is golang.org/x/net/publicsuffix, which this module needs for
// the registrable domain of a host (package contentrules, tells first-party
// loads from third-party ones) without its import of net/http/cookiejar,
// 2.2 MB of every app on Windows. The code and the data are the package's,
// under its license (LICENSE); go generate updates the data from
// publicsuffix.org. When golang.org/x/net is updated (go get), run
// go generate ./internal/psl too: a test fails while the data here is older
// than that package's.
package psl

import (
	"fmt"
	"net/netip"
	"strings"
)

// Version names the snapshot of publicsuffix.org's list that the data is.
func Version() string { return version }

// PublicSuffix returns the public suffix of the domain using a copy of the
// publicsuffix.org database compiled into the library.
//
// icann is whether the public suffix is managed by the Internet Corporation
// for Assigned Names and Numbers. If not, the public suffix is either a
// privately managed domain (and in practice, not a top level domain) or an
// unmanaged top level domain (and not explicitly mentioned in the
// publicsuffix.org list). For example, "foo.org" and "foo.co.uk" are ICANN
// domains, "foo.dyndns.org" is a private domain and
// "cromulent" is an unmanaged top level domain.
//
// Use cases for distinguishing ICANN domains like "foo.com" from private
// domains like "foo.appspot.com" can be found at
// https://wiki.mozilla.org/Public_Suffix_List/Use_Cases
func PublicSuffix(domain string) (publicSuffix string, icann bool) {
	if _, err := netip.ParseAddr(domain); err == nil {
		return domain, false
	}

	lo, hi := uint32(0), uint32(numTLD)
	s, suffix, icannNode, wildcard := domain, len(domain), false, false
loop:
	for {
		dot := strings.LastIndexByte(s, '.')
		if wildcard {
			icann = icannNode
			suffix = 1 + dot
		}
		if lo == hi {
			break
		}
		f := find(s[1+dot:], lo, hi)
		if f == notFound {
			break
		}

		u := uint32(nodes.get(f) >> (nodesBitsTextOffset + nodesBitsTextLength))
		icannNode = u&(1<<nodesBitsICANN-1) != 0
		u >>= nodesBitsICANN
		u = children.get(u & (1<<nodesBitsChildren - 1))
		lo = u & (1<<childrenBitsLo - 1)
		u >>= childrenBitsLo
		hi = u & (1<<childrenBitsHi - 1)
		u >>= childrenBitsHi
		switch u & (1<<childrenBitsNodeType - 1) {
		case nodeTypeNormal:
			suffix = 1 + dot
		case nodeTypeException:
			suffix = 1 + len(s)
			break loop
		}
		u >>= childrenBitsNodeType
		wildcard = u&(1<<childrenBitsWildcard-1) != 0
		if !wildcard {
			icann = icannNode
		}

		if dot == -1 {
			break
		}
		s = s[:dot]
	}
	if suffix == len(domain) {
		// If no rules match, the prevailing rule is "*".
		return domain[1+strings.LastIndexByte(domain, '.'):], icann
	}
	return domain[suffix:], icann
}

const notFound uint32 = 1<<32 - 1

// find returns the index of the node in the range [lo, hi) whose label equals
// label, or notFound if there is no such node. The range is assumed to be in
// strictly increasing node label order.
func find(label string, lo, hi uint32) uint32 {
	for lo < hi {
		mid := lo + (hi-lo)/2
		s := nodeLabel(mid)
		if s < label {
			lo = mid + 1
		} else if s == label {
			return mid
		} else {
			hi = mid
		}
	}
	return notFound
}

// nodeLabel returns the label for the i'th node.
func nodeLabel(i uint32) string {
	x := nodes.get(i)
	length := x & (1<<nodesBitsTextLength - 1)
	x >>= nodesBitsTextLength
	offset := x & (1<<nodesBitsTextOffset - 1)
	return text[offset : offset+length]
}

// EffectiveTLDPlusOne returns the effective top level domain plus one more
// label. For example, the eTLD+1 for "foo.bar.golang.org" is "golang.org".
func EffectiveTLDPlusOne(domain string) (string, error) {
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") || strings.Contains(domain, "..") {
		return "", fmt.Errorf("publicsuffix: empty label in domain %q", domain)
	}

	suffix, _ := PublicSuffix(domain)
	if len(domain) <= len(suffix) {
		return "", fmt.Errorf("publicsuffix: cannot derive eTLD+1 for domain %q", domain)
	}
	i := len(domain) - len(suffix) - 1
	if domain[i] != '.' {
		return "", fmt.Errorf("publicsuffix: invalid public suffix %q for domain %q", suffix, domain)
	}
	return domain[1+strings.LastIndexByte(domain[:i], '.'):], nil
}

type uint32String string

func (u uint32String) get(i uint32) uint32 {
	off := i * 4
	u = u[off:] // help the compiler reduce bounds checks
	return uint32(u[3]) |
		uint32(u[2])<<8 |
		uint32(u[1])<<16 |
		uint32(u[0])<<24
}

type uint40String string

func (u uint40String) get(i uint32) uint64 {
	off := uint64(i * (nodesBits / 8))
	u = u[off:] // help the compiler reduce bounds checks
	return uint64(u[4]) |
		uint64(u[3])<<8 |
		uint64(u[2])<<16 |
		uint64(u[1])<<24 |
		uint64(u[0])<<32
}
