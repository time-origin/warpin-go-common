package ysepay

// Some PFX files in Ysepay's official Java demo use BER indefinite-length
// encoding. Go's PKCS#12 decoders intentionally accept DER only. This file
// performs a bounded BER-to-DER conversion and recomputes the authenticated
// safe MAC with the caller-supplied password before normal PKCS#12 decoding.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"hash"
	"math/big"
	"unicode/utf16"
)

const (
	maxPKCS12Bytes = 16 << 20
	maxBERDepth    = 64
)

var (
	oidSHA1   = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
	oidSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	bigOne    = big.NewInt(1)
)

type pfxPDU struct {
	Version  int
	AuthSafe pfxContentInfo
	MacData  pfxMACData `asn1:"optional"`
}

type pfxContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"tag:0,explicit,optional"`
}

type pfxMACData struct {
	Mac        pfxDigestInfo
	MacSalt    []byte
	Iterations int `asn1:"optional,default:1"`
}

type pfxDigestInfo struct {
	Algorithm pkix.AlgorithmIdentifier
	Digest    []byte
}

type berElement struct {
	tag         []byte
	constructed bool
	indefinite  bool
	value       []byte
	children    []*berElement
}

func normalizePKCS12BER(input []byte, password string) ([]byte, error) {
	if len(input) == 0 || len(input) > maxPKCS12Bytes {
		return nil, errors.New("invalid PKCS#12 size")
	}
	outerDER, changed, err := normalizeBERDocument(input, false)
	if err != nil || !changed {
		return nil, errors.New("PKCS#12 is not a supported BER document")
	}
	var pfx pfxPDU
	if trailing, err := asn1.Unmarshal(outerDER, &pfx); err != nil || len(trailing) != 0 {
		return nil, errors.New("cannot parse normalized PKCS#12")
	}
	var authenticatedSafe []byte
	if trailing, err := asn1.Unmarshal(pfx.AuthSafe.Content.Bytes, &authenticatedSafe); err != nil || len(trailing) != 0 {
		return nil, errors.New("cannot parse PKCS#12 authenticated safe")
	}
	normalizedSafe, _, err := normalizeBERDocument(authenticatedSafe, true)
	if err != nil {
		return nil, errors.New("cannot normalize PKCS#12 authenticated safe")
	}
	encodedPassword, err := encodePKCS12Password(password)
	if err != nil {
		return nil, err
	}
	if err := verifyPKCS12MAC(&pfx.MacData, authenticatedSafe, encodedPassword); err != nil {
		return nil, err
	}
	if err := recomputePKCS12MAC(&pfx.MacData, normalizedSafe, encodedPassword); err != nil {
		return nil, err
	}
	octets, err := asn1.Marshal(normalizedSafe)
	if err != nil {
		return nil, err
	}
	pfx.AuthSafe.Content = asn1.RawValue{Class: 2, Tag: 0, IsCompound: true, Bytes: octets}
	return asn1.Marshal(pfx)
}

func verifyPKCS12MAC(macData *pfxMACData, message, password []byte) error {
	copy := *macData
	copy.Mac.Digest = append([]byte(nil), macData.Mac.Digest...)
	if err := recomputePKCS12MAC(&copy, message, password); err != nil {
		return err
	}
	if !hmac.Equal(copy.Mac.Digest, macData.Mac.Digest) {
		return errors.New("PKCS#12 password or MAC is invalid")
	}
	return nil
}

func normalizeBERDocument(input []byte, normalizeEmbedded bool) ([]byte, bool, error) {
	element, next, err := parseBERElement(input, 0, 0)
	if err != nil || next != len(input) {
		return nil, false, errors.New("invalid BER document")
	}
	encoded, nestedChanged, err := element.encode(normalizeEmbedded)
	if err != nil {
		return nil, false, err
	}
	return encoded, element.indefinite || nestedChanged || !bytes.Equal(encoded, input), nil
}

func parseBERElement(input []byte, offset, depth int) (*berElement, int, error) {
	if depth > maxBERDepth || offset >= len(input) {
		return nil, 0, errors.New("BER depth or offset is invalid")
	}
	start := offset
	first := input[offset]
	offset++
	if first&0x1f == 0x1f {
		for {
			if offset >= len(input) || offset-start > 8 {
				return nil, 0, errors.New("BER high tag is invalid")
			}
			value := input[offset]
			offset++
			if value&0x80 == 0 {
				break
			}
		}
	}
	tag := append([]byte(nil), input[start:offset]...)
	if offset >= len(input) {
		return nil, 0, errors.New("BER length is missing")
	}
	lengthByte := input[offset]
	offset++
	indefinite := lengthByte == 0x80
	length := 0
	if !indefinite {
		if lengthByte&0x80 == 0 {
			length = int(lengthByte)
		} else {
			count := int(lengthByte & 0x7f)
			if count == 0 || count > 8 || offset+count > len(input) || input[offset] == 0 {
				return nil, 0, errors.New("BER length is invalid")
			}
			for _, value := range input[offset : offset+count] {
				if length > (maxPKCS12Bytes-int(value))/256 {
					return nil, 0, errors.New("BER length overflows")
				}
				length = length*256 + int(value)
			}
			offset += count
		}
	}
	constructed := first&0x20 != 0
	if indefinite && !constructed {
		return nil, 0, errors.New("primitive BER value has indefinite length")
	}
	element := &berElement{tag: tag, constructed: constructed, indefinite: indefinite}
	if !constructed {
		if length > maxPKCS12Bytes || offset+length > len(input) {
			return nil, 0, errors.New("BER value exceeds input")
		}
		element.value = append([]byte(nil), input[offset:offset+length]...)
		return element, offset + length, nil
	}
	end := offset + length
	if !indefinite && (length > maxPKCS12Bytes || end > len(input)) {
		return nil, 0, errors.New("BER constructed value exceeds input")
	}
	for {
		if indefinite {
			if offset+2 > len(input) {
				return nil, 0, errors.New("BER indefinite value is unterminated")
			}
			if input[offset] == 0 && input[offset+1] == 0 {
				return element, offset + 2, nil
			}
		} else if offset == end {
			return element, offset, nil
		} else if offset > end {
			return nil, 0, errors.New("BER child exceeds parent")
		}
		child, next, err := parseBERElement(input, offset, depth+1)
		if err != nil {
			return nil, 0, err
		}
		element.children = append(element.children, child)
		offset = next
	}
}

func (element *berElement) encode(normalizeEmbedded bool) ([]byte, bool, error) {
	if element.isUniversalOctetString() && element.constructed {
		content, changed, err := element.flattenOctets(normalizeEmbedded)
		if err != nil {
			return nil, false, err
		}
		return encodeDERTLV([]byte{0x04}, content), true || changed, nil
	}
	// BER permits an IMPLICIT [0] OCTET STRING to be chunked. Ysepay's
	// Java-generated PFX does this for EncryptedContent. Multiple octet
	// children distinguish it from the single-value EXPLICIT [0] wrappers.
	if normalizeEmbedded && element.isContextZero() && len(element.children) > 1 && element.hasOnlyOctetChildren() {
		content, _, err := element.flattenOctets(false)
		if err != nil {
			return nil, false, err
		}
		return encodeDERTLV([]byte{0x80}, content), true, nil
	}
	if !element.constructed {
		content := element.value
		changed := false
		if normalizeEmbedded && element.isUniversalOctetString() && len(content) > 1 {
			if normalized, nestedChanged, err := normalizeBERDocument(content, true); err == nil && nestedChanged {
				content = normalized
				changed = true
			}
		}
		return encodeDERTLV(element.tag, content), changed || element.indefinite, nil
	}
	var content bytes.Buffer
	changed := element.indefinite
	for _, child := range element.children {
		encoded, childChanged, err := child.encode(normalizeEmbedded)
		if err != nil {
			return nil, false, err
		}
		content.Write(encoded)
		changed = changed || childChanged
	}
	return encodeDERTLV(element.tag, content.Bytes()), changed, nil
}

func (element *berElement) flattenOctets(normalizeEmbedded bool) ([]byte, bool, error) {
	var content bytes.Buffer
	changed := element.indefinite
	for _, child := range element.children {
		if !child.isUniversalOctetString() {
			return nil, false, errors.New("constructed OCTET STRING has a non-octet child")
		}
		if child.constructed {
			value, childChanged, err := child.flattenOctets(normalizeEmbedded)
			if err != nil {
				return nil, false, err
			}
			content.Write(value)
			changed = changed || childChanged
		} else {
			content.Write(child.value)
		}
	}
	value := content.Bytes()
	if normalizeEmbedded && len(value) > 1 {
		if normalized, nestedChanged, err := normalizeBERDocument(value, true); err == nil && nestedChanged {
			value = normalized
			changed = true
		}
	}
	return value, changed, nil
}

func (element *berElement) isUniversalOctetString() bool {
	return len(element.tag) == 1 && element.tag[0]&0xc0 == 0 && element.tag[0]&0x1f == 4
}

func (element *berElement) isContextZero() bool {
	return len(element.tag) == 1 && element.tag[0]&0xc0 == 0x80 && element.tag[0]&0x1f == 0
}

func (element *berElement) hasOnlyOctetChildren() bool {
	for _, child := range element.children {
		if !child.isUniversalOctetString() {
			return false
		}
	}
	return true
}

func encodeDERTLV(tag, content []byte) []byte {
	result := make([]byte, 0, len(tag)+9+len(content))
	result = append(result, tag...)
	if len(content) < 128 {
		result = append(result, byte(len(content)))
	} else {
		length := len(content)
		var encoded [8]byte
		index := len(encoded)
		for length > 0 {
			index--
			encoded[index] = byte(length)
			length >>= 8
		}
		result = append(result, 0x80|byte(len(encoded)-index))
		result = append(result, encoded[index:]...)
	}
	return append(result, content...)
}

func recomputePKCS12MAC(macData *pfxMACData, message, password []byte) error {
	iterations := macData.Iterations
	if iterations == 0 {
		iterations = 1
	}
	if iterations < 1 || iterations > 10_000_000 || len(macData.MacSalt) == 0 {
		return errors.New("unsupported PKCS#12 MAC parameters")
	}
	var newHash func() hash.Hash
	var sum func([]byte) []byte
	var outputSize, blockSize int
	switch {
	case macData.Mac.Algorithm.Algorithm.Equal(oidSHA1):
		newHash = sha1.New
		sum = func(input []byte) []byte { value := sha1.Sum(input); return value[:] }
		outputSize, blockSize = sha1.Size, sha1.BlockSize
	case macData.Mac.Algorithm.Algorithm.Equal(oidSHA256):
		newHash = sha256.New
		sum = func(input []byte) []byte { value := sha256.Sum256(input); return value[:] }
		outputSize, blockSize = sha256.Size, sha256.BlockSize
	default:
		return errors.New("unsupported PKCS#12 MAC algorithm")
	}
	key := derivePKCS12Key(sum, outputSize, blockSize, macData.MacSalt, password, iterations, 3, outputSize)
	mac := hmac.New(newHash, key)
	_, _ = mac.Write(message)
	macData.Mac.Digest = mac.Sum(nil)
	return nil
}

func derivePKCS12Key(sum func([]byte) []byte, outputSize, blockSize int, salt, password []byte, iterations int, purpose byte, size int) []byte {
	diversifier := bytes.Repeat([]byte{purpose}, blockSize)
	combined := append(repeatToBlock(salt, blockSize), repeatToBlock(password, blockSize)...)
	blocks := (size + outputSize - 1) / outputSize
	result := make([]byte, blocks*outputSize)
	for block := 0; block < blocks; block++ {
		digest := sum(append(append([]byte(nil), diversifier...), combined...))
		for round := 1; round < iterations; round++ {
			digest = sum(digest)
		}
		copy(result[block*outputSize:], digest)
		if block == blocks-1 {
			continue
		}
		repeatedDigest := repeatToBlock(digest, blockSize)
		increment := new(big.Int).Add(new(big.Int).SetBytes(repeatedDigest), bigOne)
		for offset := 0; offset < len(combined); offset += blockSize {
			value := new(big.Int).SetBytes(combined[offset : offset+blockSize])
			value.Add(value, increment)
			encoded := value.Bytes()
			if len(encoded) > blockSize {
				encoded = encoded[len(encoded)-blockSize:]
			}
			for index := offset; index < offset+blockSize; index++ {
				combined[index] = 0
			}
			copy(combined[offset+blockSize-len(encoded):offset+blockSize], encoded)
		}
	}
	return result[:size]
}

func repeatToBlock(pattern []byte, blockSize int) []byte {
	if len(pattern) == 0 {
		return nil
	}
	length := blockSize * ((len(pattern) + blockSize - 1) / blockSize)
	return bytes.Repeat(pattern, (length+len(pattern)-1)/len(pattern))[:length]
}

func encodePKCS12Password(password string) ([]byte, error) {
	result := make([]byte, 0, len(password)*2+2)
	for _, character := range password {
		if high, _ := utf16.EncodeRune(character); high != 0xfffd {
			return nil, fmt.Errorf("PKCS#12 password contains a non-BMP character")
		}
		result = append(result, byte(character>>8), byte(character))
	}
	return append(result, 0, 0), nil
}
