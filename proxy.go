package zxc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	tunnelConnectionEstablished = []byte("HTTP/1.1 200 Connection Established\r\n\r\n") // 通道连接建立
	internalServerErr           = []byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n\r\n", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)))
	hopToHopHeader              = []string{ // https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers
		"Keep-Alive",
		"Transfer-Encoding",
		"TE",
		"Connection",
		"Trailer",
		"Upgrade",
		"Proxy-Authorization",
		"Proxy-Authenticate",
		"Connection",
	}
)

type Proxy struct {
	delegate Delegate
	dns      *Dns
}

func New() *Proxy {
	return &Proxy{delegate: &DefaultDelegate{}, dns: &DefaultDns}
}

func NewWithDelegate(delegate Delegate, rootCa, rootKey string) *Proxy {
	Init(rootCa, rootKey)
	return &Proxy{delegate: delegate, dns: &DefaultDns}
}

func (proxy *Proxy) AddDnsRecord(host, record string) {
	proxy.dns.Add(map[string]string{host: record})
}

func (proxy *Proxy) ServerHandler(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Hostname() == proxy.dns.SslCertHost && req.URL.Path == "/ssl" { // 安装移动端证书
		rw.Header().Add("Connection", "close")
		rw.Header().Add("Content-Type", "application/x-x509-ca-cert")
		rw.Write(GetCaCert())
		return
	}

	clientConn, err := HijackerConn(rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	switch req.Method {
	case http.MethodConnect: // https
		_, _ = clientConn.Write(tunnelConnectionEstablished)

		go proxy.handleHTTPS(clientConn, req)
	default: // todo websocket
		proxy.handleHTTP(clientConn, req)
	}
}

func (proxy *Proxy) handleHTTPS(clientConn *Connection, req *http.Request) {
	defer clientConn.Close()
	certificate, err := GetCertificate(req.URL.Host)
	if err != nil {
		proxy.Error(clientConn, err)
		return
	}

	tlsConn := tls.Server(clientConn, &tls.Config{Certificates: []tls.Certificate{certificate}})
	if err := tlsConn.Handshake(); err != nil {
		proxy.Error(tlsConn, err)
		return
	}

	_ = tlsConn.SetDeadline(time.Now().Add(60 * time.Second))
	defer tlsConn.Close()

	proxyEntity, err := NewEntity(tlsConn)
	if err != nil {
		proxy.Error(tlsConn, err)
		return
	}

	proxyEntity.SetScheme("https")
	proxyEntity.SetHost(req.URL.Host)
	proxyEntity.SetRemoteAddr(req.RemoteAddr)

	proxy.delegate.BeforeRequest(proxyEntity)

	resp, err := proxy.doRequest(proxyEntity)
	if err != nil {
		proxy.Error(tlsConn, err)
		return
	}

	defer resp.Body.Close()

	if err = proxyEntity.SetResponse(resp); err != nil {
		proxy.Error(tlsConn, err)
	}

	proxy.delegate.BeforeResponse(proxyEntity, err)
	_ = resp.Write(tlsConn)
}

func (proxy *Proxy) handleHTTP(clientConn *Connection, req *http.Request) {
	defer clientConn.Close()

	proxyEntity, err := NewEntityWithRequest(req)
	if err != nil {
		proxy.Error(clientConn, err)
		return
	}

	proxy.delegate.BeforeRequest(proxyEntity)

	resp, err := proxy.doRequest(proxyEntity)
	if err != nil {
		proxy.Error(clientConn, err)
		return
	}
	defer resp.Body.Close()

	if err = proxyEntity.SetResponse(resp); err != nil {
		proxy.Error(clientConn, err)
	}

	proxy.delegate.BeforeResponse(proxyEntity, err)
	_ = resp.Write(clientConn)

}

// 请求目标服务器
func (proxy *Proxy) doRequest(entity *Entity) (*http.Response, error) {
	removeHopHeader(entity.Request.Header)

	dialer := &net.Dialer{
		Timeout:  60 * time.Second,
		Deadline: time.Now().Add(60 * time.Second),
	}
	resp, err := (&http.Transport{
		DisableKeepAlives:     true,
		ResponseHeaderTimeout: 60 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (i net.Conn, e error) {
			addr, _ = proxy.dns.CustomDialer(addr)
			return dialer.DialContext(ctx, network, addr)
		},
	}).RoundTrip(entity.Request)
	if err != nil {
		return nil, err
	}
	removeHopHeader(resp.Header)
	return resp, nil
}

// remove hop header
func removeHopHeader(header http.Header) {
	for _, hop := range hopToHopHeader {
		if value := header.Get(hop); len(value) != 0 {
			if strings.EqualFold(hop, "Connection") {
				for _, customerHeader := range strings.Split(value, ",") {
					header.Del(strings.Trim(customerHeader, " "))
				}
			}
			header.Del(hop)
		}
	}
}

func (proxy *Proxy) Error(net net.Conn, error error) {
	proxy.delegate.ErrorLog(error)
	_, _ = net.Write(internalServerErr)
	if error != nil {
		_, _ = net.Write([]byte(error.Error()))
	}
}
