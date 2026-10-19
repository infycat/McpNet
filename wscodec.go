package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
)

func wsLookLikeJSON(b []byte) bool {
	t := bytes.TrimSpace(b)
	if len(t) == 0 {
		return false
	}
	return (t[0] == '{' || t[0] == '[') && json.Valid(t)
}

func wsDecodeMessage(wsURL string, typ int, body []byte) ([]byte, int, bool) {
	if (typ != 2 && typ != 3) || len(body) == 0 {
		return body, 0, false
	}
	if wsLookLikeJSON(body) {
		return body, 0, false
	}
	ctx := &pluginCtx{Direction: "in", URL: wsURL, Host: wsHostOf(wsURL), Body: append([]byte(nil), body...)}
	if runHook("OnWebSocket", ctx) && len(ctx.Body) > 0 {
		return ctx.Body, 1, true
	}
	return body, 0, false
}

func wsEncodeMessage(wsURL string, typ int, body []byte) ([]byte, int, bool) {
	if typ != 2 || len(body) == 0 || !wsLookLikeJSON(body) {
		return body, 0, false
	}
	ctx := &pluginCtx{Direction: "out", URL: wsURL, Host: wsHostOf(wsURL), Body: append([]byte(nil), body...)}
	if runHook("OnWebSocket", ctx) && len(ctx.Body) > 0 {
		return ctx.Body, 2, true
	}
	return body, 0, false
}

// ---- XXTEA ----

func xxteaEncrypt(data []byte, key []byte) []byte {
	if len(data) == 0 {
		return data
	}
	n := (len(data) + 3) / 4
	v := make([]uint32, n)
	for i := 0; i < n; i++ {
		var value uint32
		for j := 0; j < 4; j++ {
			idx := i*4 + j
			if idx < len(data) {
				value |= uint32(data[idx]) << uint(j*8)
			}
		}
		v[i] = value
	}
	k := [4]uint32{}
	for a := 0; a < 4; a++ {
		for b := 0; b < 4; b++ {
			idx := a*4 + b
			if idx < len(key) {
				k[a] |= uint32(key[idx]) << uint(b*8)
			}
		}
	}
	const delta uint32 = 0x9E3779B9
	rounds := 6 + 52/n
	var sum uint32
	y := v[0]
	z := v[n-1]
	for r := 0; r < rounds; r++ {
		sum += delta
		e := (sum >> 2) & 3
		for p := 0; p < n-1; p++ {
			y = v[p+1]
			mx := ((z >> 5) ^ (y << 2)) + ((y >> 3) ^ (z << 4))
			mx ^= (sum ^ y) + (k[(p&3)^int(e)] ^ z)
			z = v[p] + mx
			v[p] = z
		}
		y = v[0]
		mx := ((z >> 5) ^ (y << 2)) + ((y >> 3) ^ (z << 4))
		mx ^= (sum ^ y) + (k[((n-1)&3)^int(e)] ^ z)
		z = v[n-1] + mx
		v[n-1] = z
	}
	out := make([]byte, n*4)
	for i := 0; i < n; i++ {
		for j := 0; j < 4; j++ {
			out[i*4+j] = byte(v[i] >> uint(j*8))
		}
	}
	return out
}

func xxteaDecrypt(data []byte, key []byte) []byte {
	if len(data) == 0 {
		return data
	}
	n := (len(data) + 3) / 4
	v := make([]uint32, n)
	for i := 0; i < n; i++ {
		var value uint32
		for j := 0; j < 4; j++ {
			idx := i*4 + j
			if idx < len(data) {
				value |= uint32(data[idx]) << uint(j*8)
			}
		}
		v[i] = value
	}
	k := [4]uint32{}
	for a := 0; a < 4; a++ {
		for b := 0; b < 4; b++ {
			idx := a*4 + b
			if idx < len(key) {
				k[a] |= uint32(key[idx]) << uint(b*8)
			}
		}
	}
	const delta uint32 = 0x9E3779B9
	rounds := 6 + 52/n
	sum := uint32(rounds) * delta
	y := v[0]
	z := v[n-1]
	for r := 0; r < rounds; r++ {
		e := (sum >> 2) & 3
		for p := n - 1; p > 0; p-- {
			z = v[p-1]
			mx := ((z >> 5) ^ (y << 2)) + ((y >> 3) ^ (z << 4))
			mx ^= (sum ^ y) + (k[(p&3)^int(e)] ^ z)
			y = v[p] - mx
			v[p] = y
		}
		z = v[n-1]
		mx := ((z >> 5) ^ (y << 2)) + ((y >> 3) ^ (z << 4))
		mx ^= (sum ^ y) + (k[0^int(e)] ^ z)
		y = v[0] - mx
		v[0] = y
		sum -= delta
	}
	out := make([]byte, n*4)
	for i := 0; i < n; i++ {
		for j := 0; j < 4; j++ {
			out[i*4+j] = byte(v[i] >> uint(j*8))
		}
	}
	return out
}

// ---- MessagePack subset ----

func mpEncode(v any) []byte {
	switch x := v.(type) {
	case nil:
		return []byte{0xC0}
	case bool:
		if x {
			return []byte{0xC3}
		}
		return []byte{0xC2}
	case string:
		b := []byte(x)
		switch {
		case len(b) <= 31:
			return append([]byte{0xA0 | byte(len(b))}, b...)
		case len(b) <= 0xFF:
			return append([]byte{0xD9, byte(len(b))}, b...)
		case len(b) <= 0xFFFF:
			return append([]byte{0xDA, byte(len(b) >> 8), byte(len(b))}, b...)
		default:
			return append([]byte{0xDB, byte(len(b) >> 24), byte(len(b) >> 16), byte(len(b) >> 8), byte(len(b))}, b...)
		}
	case float64:
		if x == float64(int64(x)) {
			return mpEncode(int64(x))
		}
		return []byte{0xCB, 0, 0, 0, 0, 0, 0, 0, 0}
	case int:
		return mpEncode(int64(x))
	case int64:
		if x >= 0 {
			switch {
			case x <= 0x7F:
				return []byte{byte(x)}
			case x <= 0xFF:
				return []byte{0xCC, byte(x)}
			case x <= 0xFFFF:
				return []byte{0xCD, byte(x >> 8), byte(x)}
			case x <= 0xFFFFFFFF:
				return []byte{0xCE, byte(x >> 24), byte(x >> 16), byte(x >> 8), byte(x)}
			}
			return []byte{0xCF, 0, 0, 0, 0, byte(x >> 24), byte(x >> 16), byte(x >> 8), byte(x)}
		}
		switch {
		case x >= -32:
			return []byte{byte(x)}
		case x >= -128:
			return []byte{0xD0, byte(x)}
		case x >= -32768:
			return []byte{0xD1, byte(x >> 8), byte(x)}
		case x >= -2147483648:
			return []byte{0xD2, byte(x >> 24), byte(x >> 16), byte(x >> 8), byte(x)}
		}
		return []byte{0xD3, 0, 0, 0, 0, byte(x >> 24), byte(x >> 16), byte(x >> 8), byte(x)}
	case []any:
		var body bytes.Buffer
		for _, it := range x {
			body.Write(mpEncode(it))
		}
		var head []byte
		switch {
		case len(x) <= 15:
			head = []byte{0x90 | byte(len(x))}
		case len(x) <= 0xFFFF:
			head = []byte{0xDC, byte(len(x) >> 8), byte(len(x))}
		default:
			head = []byte{0xDD, byte(len(x) >> 24), byte(len(x) >> 16), byte(len(x) >> 8), byte(len(x))}
		}
		return append(head, body.Bytes()...)
	case map[string]any:
		var body bytes.Buffer
		for k, val := range x {
			body.Write(mpEncode(k))
			body.Write(mpEncode(val))
		}
		var head []byte
		switch {
		case len(x) <= 15:
			head = []byte{0x80 | byte(len(x))}
		case len(x) <= 0xFFFF:
			head = []byte{0xDE, byte(len(x) >> 8), byte(len(x))}
		default:
			head = []byte{0xDF, byte(len(x) >> 24), byte(len(x) >> 16), byte(len(x) >> 8), byte(len(x))}
		}
		return append(head, body.Bytes()...)
	default:
		return []byte{0xC0}
	}
}

type mpReader struct {
	b []byte
	i int
}

func (r *mpReader) byte() byte {
	if r.i >= len(r.b) {
		return 0
	}
	v := r.b[r.i]
	r.i++
	return v
}

func (r *mpReader) take(n int) []byte {
	if n < 0 || r.i+n > len(r.b) {
		n = len(r.b) - r.i
	}
	v := r.b[r.i : r.i+n]
	r.i += n
	return v
}

func mpDecode(data []byte) (any, error) {
	r := &mpReader{b: data}
	return r.next()
}

func (r *mpReader) next() (any, error) {
	if r.i >= len(r.b) {
		return nil, fmt.Errorf("messagepack: unexpected end")
	}
	f := r.byte()
	switch {
	case f <= 0x7F:
		return int64(f), nil
	case f >= 0xE0:
		return int64(int8(f)), nil
	case f >= 0xA0 && f <= 0xBF:
		return string(r.take(int(f - 0xA0))), nil
	case f >= 0x90 && f <= 0x9F:
		n := int(f - 0x90)
		arr := make([]any, 0, n)
		for i := 0; i < n; i++ {
			v, err := r.next()
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		return arr, nil
	case f >= 0x80 && f <= 0x8F:
		n := int(f - 0x80)
		m := make(map[string]any, n)
		for i := 0; i < n; i++ {
			k, err := r.next()
			if err != nil {
				return nil, err
			}
			v, err := r.next()
			if err != nil {
				return nil, err
			}
			m[fmt.Sprint(k)] = v
		}
		return m, nil
	}
	switch f {
	case 0xC0:
		return nil, nil
	case 0xC2:
		return false, nil
	case 0xC3:
		return true, nil
	case 0xCA:
		r.take(4)
		return float64(0), nil
	case 0xCB:
		r.take(8)
		return float64(0), nil
	case 0xCC:
		return int64(r.byte()), nil
	case 0xCD:
		return int64(binary.BigEndian.Uint16(r.take(2))), nil
	case 0xCE:
		return int64(binary.BigEndian.Uint32(r.take(4))), nil
	case 0xCF:
		r.take(4)
		return int64(binary.BigEndian.Uint32(r.take(4))), nil
	case 0xD0:
		return int64(int8(r.byte())), nil
	case 0xD1:
		return int64(int16(binary.BigEndian.Uint16(r.take(2)))), nil
	case 0xD2:
		return int64(int32(binary.BigEndian.Uint32(r.take(4)))), nil
	case 0xD3:
		r.take(4)
		return int64(int32(binary.BigEndian.Uint32(r.take(4)))), nil
	case 0xD9:
		return string(r.take(int(r.byte()))), nil
	case 0xDA:
		return string(r.take(int(binary.BigEndian.Uint16(r.take(2))))), nil
	case 0xDB:
		return string(r.take(int(binary.BigEndian.Uint32(r.take(4))))), nil
	case 0xDC:
		n := int(binary.BigEndian.Uint16(r.take(2)))
		arr := make([]any, 0, n)
		for i := 0; i < n; i++ {
			v, err := r.next()
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		return arr, nil
	case 0xDD:
		n := int(binary.BigEndian.Uint32(r.take(4)))
		arr := make([]any, 0, n)
		for i := 0; i < n; i++ {
			v, err := r.next()
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		return arr, nil
	case 0xDE:
		n := int(binary.BigEndian.Uint16(r.take(2)))
		m := make(map[string]any, n)
		for i := 0; i < n; i++ {
			k, err := r.next()
			if err != nil {
				return nil, err
			}
			v, err := r.next()
			if err != nil {
				return nil, err
			}
			m[fmt.Sprint(k)] = v
		}
		return m, nil
	case 0xDF:
		n := int(binary.BigEndian.Uint32(r.take(4)))
		m := make(map[string]any, n)
		for i := 0; i < n; i++ {
			k, err := r.next()
			if err != nil {
				return nil, err
			}
			v, err := r.next()
			if err != nil {
				return nil, err
			}
			m[fmt.Sprint(k)] = v
		}
		return m, nil
	}
	return nil, fmt.Errorf("messagepack: unsupported 0x%02x", f)
}

func wsEncodeTextMessage(wsURL string, typ int, body []byte) ([]byte, int, bool) {
	if typ != 2 || len(body) == 0 {
		return body, 0, false
	}
	ctx := &pluginCtx{Direction: "out", URL: wsURL, Host: wsHostOf(wsURL), Body: append([]byte(nil), body...)}
	if runHook("OnWebSocket", ctx) && len(ctx.Body) > 0 {
		return ctx.Body, 2, true
	}
	return body, 0, false
}

func xxteaPack(v any, key []byte) ([]byte, error) {
	encoded := mpEncode(v)
	total := 4 + len(encoded)
	pad := (4 - (total % 4)) % 4
	buf := make([]byte, total+pad)
	binary.BigEndian.PutUint32(buf[:4], uint32(len(encoded)))
	copy(buf[4:], encoded)
	return xxteaEncrypt(buf, key), nil
}

func xxteaUnpack(enc []byte, key []byte) (any, error) {
	if len(enc) == 0 {
		return nil, fmt.Errorf("xxtea: empty ciphertext")
	}
	if len(enc)%4 != 0 {
		return nil, fmt.Errorf("xxtea: ciphertext length must be a multiple of 4")
	}
	dec := xxteaDecrypt(enc, key)
	if len(dec) < 4 {
		return nil, fmt.Errorf("xxtea: short plaintext")
	}
	n := binary.BigEndian.Uint32(dec[:4])
	if int(n) > len(dec)-4 {
		return nil, fmt.Errorf("xxtea: invalid length header")
	}
	return mpDecode(dec[4 : 4+int(n)])
}

func wsHostOf(wsURL string) string {
	if i := strings.Index(wsURL, "://"); i >= 0 {
		rest := wsURL[i+3:]
		if j := strings.IndexAny(rest, "/?#"); j >= 0 {
			rest = rest[:j]
		}
		return rest
	}
	return wsURL
}
