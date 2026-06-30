package networking

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/quic-go/quic-go"
)

type QUICProtocol struct {
	logger        *slog.Logger
	listener      *quic.Listener
	connections   map[peer.ID]*quic.Conn
	config        QUICConfig
	messageChan   chan *NetworkMessage
	healthMetrics *ProtocolHealthMetrics

	mu        sync.RWMutex
	listenAddr string
	tlsConfig *tls.Config
	peers     map[peer.ID]string
}

type QUICConfig struct {
	ListenAddr string `json:"listen_addr"`
}

func NewQUICProtocol(logger *slog.Logger, config QUICConfig) (*QUICProtocol, error) {
	tlsConf, err := generateTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("quic tls: %w", err)
	}
	return &QUICProtocol{
		logger:        logger,
		config:        config,
		tlsConfig:     tlsConf,
		connections:   make(map[peer.ID]*quic.Conn),
		messageChan:   make(chan *NetworkMessage, 100),
		healthMetrics: &ProtocolHealthMetrics{ProtocolType: ProtocolQUIC},
		peers:         make(map[peer.ID]string),
	}, nil
}

func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{certDER},
			PrivateKey:  key,
		}},
		NextProtos: []string{"apa-quic"},
	}, nil
}

func (qp *QUICProtocol) Initialize(ctx context.Context) error {
	addr := qp.config.ListenAddr
	if addr == "" {
		addr = ":0"
	}
	listener, err := quic.ListenAddr(addr, qp.tlsConfig, nil)
	if err != nil {
		return fmt.Errorf("quic listen: %w", err)
	}
	qp.listener = listener
	qp.listenAddr = listener.Addr().String()

	go qp.acceptLoop(ctx)
	qp.logger.Info("QUIC protocol initialized", "listen_addr", qp.listenAddr)
	return nil
}

func (qp *QUICProtocol) acceptLoop(ctx context.Context) {
	go func() {
		<-ctx.Done()
		_ = qp.listener.Close()
	}()
	for {
		conn, err := qp.listener.Accept(ctx)
		if err != nil {
			return
		}
		go qp.handleConn(ctx, conn)
	}
}

func (qp *QUICProtocol) handleConn(ctx context.Context, conn *quic.Conn) {
	qp.healthMetrics.ConnectionStatus = ConnectionConnected
	defer func() { qp.healthMetrics.ConnectionStatus = ConnectionDisconnected }()

	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return
		}
		go func() {
			defer func() { _ = stream.Close() }()
			var buf bytes.Buffer
			tmp := make([]byte, 4096)
			for {
				n, err := stream.Read(tmp)
				if err != nil {
					return
				}
				buf.Write(tmp[:n])
				if n < len(tmp) {
					break
				}
			}
			var msg NetworkMessage
			if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
				return
			}
			qp.healthMetrics.TotalMessagesRecv++
			select {
			case qp.messageChan <- &msg:
			default:
			}
		}()
	}
}

func (qp *QUICProtocol) RegisterPeerEndpoint(id peer.ID, addr string) {
	qp.mu.Lock()
	qp.peers[id] = addr
	qp.mu.Unlock()
}

func (qp *QUICProtocol) SendMessage(to peer.ID, message *NetworkMessage) error {
	qp.mu.RLock()
	addr, ok := qp.peers[to]
	qp.mu.RUnlock()
	if !ok {
		return fmt.Errorf("quic: unknown peer %s", to)
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"apa-quic"},
	}
	conn, err := quic.DialAddr(context.Background(), addr, tlsConf, nil)
	if err != nil {
		return fmt.Errorf("quic dial: %w", err)
	}
	defer func() { _ = conn.CloseWithError(0, "") }()

	stream, err := conn.OpenStream()
	if err != nil {
		return fmt.Errorf("quic open stream: %w", err)
	}
	defer func() { _ = stream.Close() }()

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("quic marshal: %w", err)
	}
	if _, err := stream.Write(data); err != nil {
		return fmt.Errorf("quic write: %w", err)
	}
	qp.healthMetrics.TotalMessagesSent++
	return nil
}

func (qp *QUICProtocol) ReceiveMessages() <-chan *NetworkMessage {
	return qp.messageChan
}

func (qp *QUICProtocol) GetConnectionInfo() *ConnectionInfo {
	status := qp.healthMetrics.ConnectionStatus
	if status == "" {
		status = ConnectionConnected
	}
	return &ConnectionInfo{
		LocalAddress: qp.listenAddr,
		Protocol:     ProtocolQUIC,
		Status:       status,
		Established:  time.Now(),
		LastActivity: time.Now(),
	}
}

func (qp *QUICProtocol) GetHealthMetrics() *ProtocolHealthMetrics {
	return qp.healthMetrics
}

func (qp *QUICProtocol) Close() error {
	if qp.listener != nil {
		return qp.listener.Close()
	}
	return nil
}
