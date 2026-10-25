package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const maxMCPNetworkTimeout = 60 * time.Second

const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func mcpRequestTimeout(args map[string]any, defaultTimeout time.Duration) (time.Duration, error) {
	ms := argInt(args, "timeout_ms", int(defaultTimeout/time.Millisecond))
	if ms < 100 || time.Duration(ms)*time.Millisecond > maxMCPNetworkTimeout {
		return 0, fmt.Errorf("timeout_ms must be between 100 and %d", int(maxMCPNetworkTimeout/time.Millisecond))
	}
	return time.Duration(ms) * time.Millisecond, nil
}

func mcpRequestHeaders(value any) (http.Header, []string, error) {
	headers := make(http.Header)
	input, _ := value.(map[string]any)
	if len(input) > 64 {
		return nil, nil, fmt.Errorf("at most 64 request headers are allowed")
	}
	names := make([]string, 0, len(input))
	totalBytes := 0
	for name, raw := range input {
		name = strings.TrimSpace(name)
		fieldValue, ok := raw.(string)
		if !ok || name == "" || strings.ContainsAny(name, "\r\n:") || strings.ContainsAny(fieldValue, "\r\n") {
			return nil, nil, fmt.Errorf("headers must map valid header names to single-line string values")
		}
		if strings.EqualFold(name, "Host") || strings.EqualFold(name, "Content-Length") {
			return nil, nil, fmt.Errorf("Host and Content-Length are controlled by the HTTP transport")
		}
		if len(name) > 256 || len(fieldValue) > 8192 {
			return nil, nil, fmt.Errorf("header name or value is too long")
		}
		totalBytes += len(name) + len(fieldValue)
		if totalBytes > 64<<10 {
			return nil, nil, fmt.Errorf("request headers exceed 64 KiB")
		}
		headers.Set(name, fieldValue)
		names = append(names, http.CanonicalHeaderKey(name))
	}
	sort.Strings(names)
	return headers, names, nil
}

func (s *mcpServer) sendHTTPMCPRequest(args map[string]any) (map[string]any, error) {
	rawURL := strings.TrimSpace(stringArg(args, "url"))
	if len(rawURL) > 16<<10 {
		return nil, fmt.Errorf("url exceeds the 16 KiB limit")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return nil, fmt.Errorf("url must be an absolute http:// or https:// URL without embedded credentials")
	}
	method := strings.TrimSpace(stringArg(args, "method"))
	if method == "" {
		method = http.MethodGet
	}
	if len(method) > 32 || strings.ContainsAny(method, " \t\r\n") {
		return nil, fmt.Errorf("method must be a valid HTTP method token")
	}
	bodyText := stringArg(args, "body_base64")
	body, err := base64.StdEncoding.DecodeString(bodyText)
	if err != nil {
		return nil, fmt.Errorf("body_base64 is invalid")
	}
	if len(body) > maxMCPBodyBytes {
		return nil, fmt.Errorf("request body exceeds the %d-byte limit", maxMCPBodyBytes)
	}
	headers, headerNames, err := mcpRequestHeaders(args["headers"])
	if err != nil {
		return nil, err
	}
	timeout, err := mcpRequestTimeout(args, 10*time.Second)
	if err != nil {
		return nil, err
	}
	includeResponse := true
	if _, provided := args["include_response"]; provided {
		includeResponse = argBool(args, "include_response")
	}
	safeTarget := redactCaptureURL(parsed.String())
	if includeResponse && !s.authorize("private_read", "McpNet HTTP response data", fmt.Sprintf("AI 客戶端要求傳送 HTTP 請求至 %s，並將回應標頭與最多 %d bytes 的回應本文分享給 AI。回應可能包含個人或憑證資料。允許？\n\nAn AI client requests an HTTP request to %s and asks to share response headers and up to %d bytes of response body with the AI. The response may contain personal or credential data. Allow?", safeTarget, maxMCPBodyBytes, safeTarget, maxMCPBodyBytes)) {
		return nil, fmt.Errorf("user denied sharing the HTTP response")
	}
	if !s.authorize("network_write", "McpNet HTTP request", fmt.Sprintf("AI 客戶端要求向 %s 發送 %s 請求（%d bytes；標頭：%s）。這會建立新的網路連線；回應本文%s會回傳給 AI。允許？\n\nAn AI client requests a new %s HTTP connection to %s (%d body bytes; headers: %s). This sends a new network request; the response body %s be returned to the AI. Allow?", safeTarget, method, len(body), strings.Join(headerNames, ", "), map[bool]string{true: " and headers ", false: "will not "}[includeResponse], method, safeTarget, len(body), strings.Join(headerNames, ", "), map[bool]string{true: "and headers will", false: "will not"}[includeResponse])) {
		return nil, fmt.Errorf("user denied the HTTP request")
	}

	request, err := http.NewRequest(method, parsed.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP request method or target")
	}
	request.Header = headers
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil // MCP requests go only to the explicitly authorized target.
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer transport.CloseIdleConnections()
	started := time.Now()
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("HTTP request to %s failed (network or TLS error; URL details omitted)", safeTarget)
	}
	defer response.Body.Close()
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, maxMCPBodyBytes+1))
	if readErr != nil {
		return nil, fmt.Errorf("HTTP response read failed (details omitted)")
	}
	truncated := len(responseBody) > maxMCPBodyBytes
	if truncated {
		responseBody = responseBody[:maxMCPBodyBytes]
	}
	result := map[string]any{
		"method": method, "url": safeTarget, "status_code": response.StatusCode,
		"status": response.Status, "request_body_bytes": len(body),
		"response_body_bytes": len(responseBody), "response_body_truncated": truncated,
		"duration_ms": time.Since(started).Milliseconds(),
	}
	if includeResponse {
		result["response_body_base64"] = base64.StdEncoding.EncodeToString(responseBody)
		result["response_headers"] = response.Header
	}
	return result, nil
}

func (s *mcpServer) sendSocketMCPRequest(args map[string]any) (map[string]any, error) {
	protocol := strings.ToLower(strings.TrimSpace(stringArg(args, "protocol")))
	if protocol != "tcp" && protocol != "udp" {
		return nil, fmt.Errorf("protocol must be tcp or udp")
	}
	address := strings.TrimSpace(stringArg(args, "address"))
	if len(address) > 1024 {
		return nil, fmt.Errorf("address exceeds the 1024-byte limit")
	}
	host, portText, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return nil, fmt.Errorf("address must be a host:port value")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("address port must be between 1 and 65535")
	}
	data, err := base64.StdEncoding.DecodeString(stringArg(args, "data_base64"))
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("data_base64 must contain a non-empty Base64 payload")
	}
	if len(data) > maxMCPBodyBytes {
		return nil, fmt.Errorf("request payload exceeds the %d-byte limit", maxMCPBodyBytes)
	}
	useTLS := argBool(args, "use_tls")
	if useTLS && protocol != "tcp" {
		return nil, fmt.Errorf("TLS is available only for TCP connections")
	}
	serverName := strings.TrimSpace(stringArg(args, "server_name"))
	if len(serverName) > 253 || strings.ContainsAny(serverName, "\r\n\x00") {
		return nil, fmt.Errorf("server_name must be a valid host name of at most 253 bytes")
	}
	if useTLS && serverName == "" {
		serverName = host
	}
	timeout, err := mcpRequestTimeout(args, 5*time.Second)
	if err != nil {
		return nil, err
	}
	readResponse := true
	if _, provided := args["read_response"]; provided {
		readResponse = argBool(args, "read_response")
	}
	if readResponse && !s.authorize("private_read", "McpNet socket response data", fmt.Sprintf("AI 客戶端要求向 %s://%s 發送 %d bytes，並將最多 %d bytes 回應資料分享給 AI。回應可能含個人或憑證資料。允許？\n\nAn AI client requests sending %d bytes to %s://%s and sharing up to %d response bytes with the AI. The response may contain personal or credential data. Allow?", protocol, address, len(data), maxMCPBodyBytes, len(data), protocol, address, maxMCPBodyBytes)) {
		return nil, fmt.Errorf("user denied sharing the socket response")
	}
	if !s.authorize("network_write", "McpNet socket request", fmt.Sprintf("AI 客戶端要求建立新的 %s 連線至 %s（TLS SNI／驗證名稱：%s），發送 %d bytes%s。允許？\n\nAn AI client requests a new %s connection to %s (TLS SNI/verification name: %s), sending %d bytes%s. Allow?", protocol, address, serverName, len(data), map[bool]string{true: "並讀取回應", false: ""}[readResponse], protocol, address, serverName, len(data), map[bool]string{true: " and reads the response", false: ""}[readResponse])) {
		return nil, fmt.Errorf("user denied the socket request")
	}

	started := time.Now()
	connection, err := net.DialTimeout(protocol, address, timeout)
	if err != nil {
		return nil, fmt.Errorf("%s connection to %s failed (details omitted)", protocol, address)
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(timeout))
	if useTLS {
		tlsConnection := tls.Client(connection, &tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS12})
		if err := tlsConnection.Handshake(); err != nil {
			return nil, fmt.Errorf("TLS handshake to %s failed (details omitted)", address)
		}
		connection = tlsConnection
	}
	written := 0
	for written < len(data) {
		n, writeErr := connection.Write(data[written:])
		written += n
		if writeErr != nil {
			return nil, fmt.Errorf("socket request write failed")
		}
		if n == 0 {
			return nil, fmt.Errorf("socket request write made no progress")
		}
		if protocol == "udp" {
			break // One Write is one datagram.
		}
	}
	result := map[string]any{
		"protocol": protocol, "address": address, "tls": useTLS,
		"request_bytes": written, "duration_ms": time.Since(started).Milliseconds(),
	}
	if !readResponse {
		if closeWriter, ok := connection.(interface{ CloseWrite() error }); ok {
			_ = closeWriter.CloseWrite()
		}
		result["response_bytes"] = 0
		result["response_base64"] = ""
		result["response_truncated"] = false
		result["read_timed_out"] = false
		return result, nil
	}
	response := make([]byte, 0, 4096)
	buffer := make([]byte, 8192)
	timedOut := false
	for len(response) <= maxMCPBodyBytes {
		n, readErr := connection.Read(buffer)
		if n > 0 {
			remaining := maxMCPBodyBytes + 1 - len(response)
			if n > remaining {
				n = remaining
			}
			response = append(response, buffer[:n]...)
		}
		if protocol == "udp" || readErr != nil {
			if netErr, ok := readErr.(net.Error); ok && netErr.Timeout() {
				timedOut = true
			}
			if readErr != nil && readErr != io.EOF && !timedOut {
				return nil, fmt.Errorf("socket response read failed")
			}
			break
		}
	}
	truncated := len(response) > maxMCPBodyBytes
	if truncated {
		response = response[:maxMCPBodyBytes]
	}
	result["response_bytes"] = len(response)
	result["response_base64"] = base64.StdEncoding.EncodeToString(response)
	result["response_truncated"] = truncated
	result["read_timed_out"] = timedOut
	result["duration_ms"] = time.Since(started).Milliseconds()
	return result, nil
}

func (s *mcpServer) sendWebSocketMCPRequest(args map[string]any) (map[string]any, error) {
	rawURL := strings.TrimSpace(stringArg(args, "url"))
	if len(rawURL) > 16<<10 {
		return nil, fmt.Errorf("url exceeds the 16 KiB limit")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "ws" && parsed.Scheme != "wss") || parsed.User != nil || parsed.Fragment != "" {
		return nil, fmt.Errorf("url must be an absolute ws:// or wss:// URL without credentials or fragments")
	}
	payload, err := base64.StdEncoding.DecodeString(stringArg(args, "data_base64"))
	if err != nil || len(payload) == 0 {
		return nil, fmt.Errorf("data_base64 must contain a non-empty Base64 payload")
	}
	if len(payload) > maxMCPBodyBytes {
		return nil, fmt.Errorf("request payload exceeds the %d-byte limit", maxMCPBodyBytes)
	}
	messageType := strings.ToLower(strings.TrimSpace(stringArg(args, "message_type")))
	if messageType == "" {
		messageType = "text"
	}
	if messageType != "text" && messageType != "binary" {
		return nil, fmt.Errorf("message_type must be text or binary")
	}
	if messageType == "text" && !utf8.Valid(payload) {
		return nil, fmt.Errorf("text WebSocket payload must be valid UTF-8")
	}
	headers, headerNames, err := mcpRequestHeaders(args["headers"])
	if err != nil {
		return nil, err
	}
	for name := range headers {
		switch strings.ToLower(name) {
		case "host", "content-length", "connection", "upgrade", "sec-websocket-key", "sec-websocket-version", "sec-websocket-protocol", "sec-websocket-extensions":
			return nil, fmt.Errorf("header %q is managed by the WebSocket handshake", name)
		}
	}
	timeout, err := mcpRequestTimeout(args, 5*time.Second)
	if err != nil {
		return nil, err
	}
	readResponse := true
	if _, provided := args["read_response"]; provided {
		readResponse = argBool(args, "read_response")
	}
	safeTarget := redactCaptureURL(parsed.String())
	if readResponse && !s.authorize("private_read", "McpNet WebSocket response data", fmt.Sprintf("AI 客戶端要求連線至 %s，發送一則 %s 訊息，並將最多 %d bytes 的回應訊息分享給 AI。資料可能含個人或憑證資訊。允許？\n\nAn AI client requests a WebSocket connection to %s, sending one %s message and sharing up to %d response bytes with the AI. Data may contain personal or credential information. Allow?", safeTarget, messageType, maxMCPBodyBytes, safeTarget, messageType, maxMCPBodyBytes)) {
		return nil, fmt.Errorf("user denied sharing the WebSocket response")
	}
	if !s.authorize("network_write", "McpNet WebSocket request", fmt.Sprintf("AI 客戶端要求建立新的 WebSocket 連線至 %s，發送一則 %s 訊息（%d bytes；標頭：%s）%s。允許？\n\nAn AI client requests a new WebSocket connection to %s, sending one %s message (%d bytes; headers: %s)%s. Allow?", safeTarget, messageType, len(payload), strings.Join(headerNames, ", "), map[bool]string{true: "並讀取回應", false: ""}[readResponse], safeTarget, messageType, len(payload), strings.Join(headerNames, ", "), map[bool]string{true: " and reading a response", false: ""}[readResponse])) {
		return nil, fmt.Errorf("user denied the WebSocket request")
	}

	dialer := &net.Dialer{Timeout: timeout}
	started := time.Now()
	dialAddress := parsed.Host
	if parsed.Port() == "" {
		defaultPort := "80"
		if parsed.Scheme == "wss" {
			defaultPort = "443"
		}
		dialAddress = net.JoinHostPort(parsed.Hostname(), defaultPort)
	}
	connection, err := dialer.Dial("tcp", dialAddress)
	if err != nil {
		return nil, fmt.Errorf("WebSocket connection to %s failed (details omitted)", safeTarget)
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(timeout))
	if parsed.Scheme == "wss" {
		host, _, _ := net.SplitHostPort(parsed.Host)
		if host == "" {
			host = parsed.Hostname()
		}
		tlsConnection := tls.Client(connection, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		if err := tlsConnection.Handshake(); err != nil {
			return nil, fmt.Errorf("secure WebSocket handshake to %s failed (details omitted)", safeTarget)
		}
		connection = tlsConnection
	}
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("could not create a WebSocket handshake key")
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)
	requestURI := parsed.RequestURI()
	if requestURI == "" {
		requestURI = "/"
	}
	var handshake strings.Builder
	fmt.Fprintf(&handshake, "GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n", requestURI, parsed.Host, key)
	for name, values := range headers {
		for _, value := range values {
			fmt.Fprintf(&handshake, "%s: %s\r\n", name, value)
		}
	}
	handshake.WriteString("\r\n")
	if err := writeAll(connection, []byte(handshake.String())); err != nil {
		return nil, fmt.Errorf("WebSocket handshake request failed")
	}
	buffered := bufio.NewReader(connection)
	request := &http.Request{Method: http.MethodGet}
	response, err := http.ReadResponse(buffered, request)
	if err != nil {
		return nil, fmt.Errorf("WebSocket handshake response failed")
	}
	if response.StatusCode != http.StatusSwitchingProtocols || !headerHasToken(response.Header, "Connection", "upgrade") || !strings.EqualFold(strings.TrimSpace(response.Header.Get("Upgrade")), "websocket") {
		if response.Body != nil {
			_ = response.Body.Close()
		}
		return map[string]any{"url": safeTarget, "status_code": response.StatusCode, "status": response.Status, "upgraded": false, "duration_ms": time.Since(started).Milliseconds()}, nil
	}
	acceptHash := sha1.Sum([]byte(key + websocketGUID))
	expectedAccept := base64.StdEncoding.EncodeToString(acceptHash[:])
	if response.Header.Get("Sec-WebSocket-Accept") != expectedAccept {
		return nil, fmt.Errorf("WebSocket server returned an invalid handshake acceptance")
	}
	opcode := byte(1)
	if messageType == "binary" {
		opcode = 2
	}
	if err := writeWebSocketFrame(connection, opcode, payload); err != nil {
		return nil, fmt.Errorf("WebSocket message send failed")
	}
	result := map[string]any{"url": safeTarget, "status_code": response.StatusCode, "status": response.Status, "upgraded": true, "message_type": messageType, "request_bytes": len(payload), "duration_ms": time.Since(started).Milliseconds()}
	if !readResponse {
		_ = writeWebSocketFrame(connection, 8, nil)
		result["response_base64"] = ""
		result["response_bytes"] = 0
		result["response_truncated"] = false
		result["read_timed_out"] = false
		return result, nil
	}
	responseType, responseData, timedOut, closeReceived, err := readWebSocketMessage(buffered, connection)
	if err != nil {
		return nil, fmt.Errorf("WebSocket response read failed")
	}
	if timedOut && !closeReceived {
		_ = writeWebSocketFrame(connection, 8, nil)
	}
	result["response_type"] = responseType
	result["response_base64"] = base64.StdEncoding.EncodeToString(responseData)
	result["response_bytes"] = len(responseData)
	result["response_truncated"] = len(responseData) > maxMCPBodyBytes
	if len(responseData) > maxMCPBodyBytes {
		result["response_base64"] = base64.StdEncoding.EncodeToString(responseData[:maxMCPBodyBytes])
		result["response_bytes"] = maxMCPBodyBytes
	}
	result["read_timed_out"] = timedOut
	result["close_received"] = closeReceived
	result["duration_ms"] = time.Since(started).Milliseconds()
	return result, nil
}

func headerHasToken(header http.Header, name, token string) bool {
	for _, value := range header.Values(name) {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

func writeWebSocketFrame(writer io.Writer, opcode byte, payload []byte) error {
	var header []byte
	length := len(payload)
	switch {
	case length < 126:
		header = []byte{0x80 | opcode, 0x80 | byte(length)}
	case length <= 65535:
		header = make([]byte, 4)
		header[0], header[1] = 0x80|opcode, 0x80|126
		binary.BigEndian.PutUint16(header[2:], uint16(length))
	default:
		header = make([]byte, 10)
		header[0], header[1] = 0x80|opcode, 0x80|127
		binary.BigEndian.PutUint64(header[2:], uint64(length))
	}
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	header = append(header, mask...)
	masked := make([]byte, length)
	for i := range payload {
		masked[i] = payload[i] ^ mask[i&3]
	}
	if err := writeAll(writer, header); err != nil {
		return err
	}
	return writeAll(writer, masked)
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := writer.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

func readWebSocketMessage(reader *bufio.Reader, writer io.Writer) (string, []byte, bool, bool, error) {
	message := make([]byte, 0, 4096)
	messageType := ""
	var timedOut, closed bool
	for {
		first, err := reader.ReadByte()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return messageType, message, true, closed, nil
			}
			if err == io.EOF {
				return messageType, message, timedOut, true, nil
			}
			return "", nil, false, false, err
		}
		second, err := reader.ReadByte()
		if err != nil {
			return "", nil, false, false, err
		}
		fin := first&0x80 != 0
		opcode := first & 0x0f
		if first&0x70 != 0 {
			return "", nil, false, false, fmt.Errorf("unsupported WebSocket extension bits")
		}
		masked := second&0x80 != 0
		length := uint64(second & 0x7f)
		if length == 126 {
			var extended [2]byte
			if _, err := io.ReadFull(reader, extended[:]); err != nil {
				return "", nil, false, false, err
			}
			length = uint64(binary.BigEndian.Uint16(extended[:]))
		} else if length == 127 {
			var extended [8]byte
			if _, err := io.ReadFull(reader, extended[:]); err != nil {
				return "", nil, false, false, err
			}
			length = binary.BigEndian.Uint64(extended[:])
			if length>>63 != 0 {
				return "", nil, false, false, fmt.Errorf("invalid WebSocket frame length")
			}
		}
		if length > maxMCPBodyBytes+125 || uint64(len(message))+length > maxMCPBodyBytes+125 {
			return "", nil, false, false, fmt.Errorf("WebSocket response exceeds the size limit")
		}
		if opcode >= 8 && (!fin || length > 125) {
			return "", nil, false, false, fmt.Errorf("invalid WebSocket control frame")
		}
		var mask [4]byte
		if masked {
			if _, err := io.ReadFull(reader, mask[:]); err != nil {
				return "", nil, false, false, err
			}
		}
		frame := make([]byte, int(length))
		if _, err := io.ReadFull(reader, frame); err != nil {
			return "", nil, false, false, err
		}
		if masked {
			for i := range frame {
				frame[i] ^= mask[i&3]
			}
		}
		switch opcode {
		case 8:
			closed = true
			_ = writeWebSocketFrame(writer, 8, frame)
			return messageType, message, timedOut, closed, nil
		case 9:
			if err := writeWebSocketFrame(writer, 10, frame); err != nil {
				return "", nil, false, false, err
			}
			continue
		case 10:
			continue
		case 1:
			if messageType != "" {
				return "", nil, false, false, fmt.Errorf("unexpected nested WebSocket message")
			}
			messageType = "text"
		case 2:
			if messageType != "" {
				return "", nil, false, false, fmt.Errorf("unexpected nested WebSocket message")
			}
			messageType = "binary"
		case 0:
			if messageType == "" {
				return "", nil, false, false, fmt.Errorf("unexpected WebSocket continuation frame")
			}
		default:
			return "", nil, false, false, fmt.Errorf("unsupported WebSocket frame opcode")
		}
		message = append(message, frame...)
		if fin && messageType != "" {
			if messageType == "text" && !utf8.Valid(message) {
				return "", nil, false, false, fmt.Errorf("WebSocket text response is not valid UTF-8")
			}
			return messageType, message, timedOut, closed, nil
		}
	}
}
