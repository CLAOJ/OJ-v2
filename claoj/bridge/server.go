package bridge

import (
	"log"
	"net"

	"github.com/CLAOJ/claoj/config"
)

// Server accepts incoming TCP connections from judge workers.
type Server struct {
	addr    string
	manager *Manager
}

func NewServer() *Server {
	addr := config.C.Bridge.Addr
	if addr == "" {
		addr = ":9999" // Default DMOJ bridge port
	}
	return &Server{
		addr:    addr,
		manager: NewManager(),
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	log.Printf("bridge: TCP server listening on %s", s.addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("bridge: accept error: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(netConn net.Conn) {
	conn := NewConnection(netConn)
	defer conn.Close()

	handler := NewHandler(conn, s.manager)
	handler.loop()
}
