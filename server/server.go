package server

import (
	"bufio"
	"crypto/tls"
	"net"
	"sync"
)

// Hook is can be added to run before every cmd. Return true if the command is done.
type Callback func(*Peer, string, ...string) bool
type Cmd func(c *Peer, cmd string, args []string)
type Peer struct {
	writer       *bufio.Writer
	closed       bool
	Resp3        bool
	Ctx          interface{} // anything goes, server won't touch this
	onDisconnect []func()    // list of callbacks
	mu           sync.Mutex  // for Block()
}

type Server struct {
	listener  net.Listener
	cmds      map[string]Cmd
	preHook   Callback
	peers     map[net.Conn]struct{}
	mu        sync.Mutex
	wg        sync.WaitGroup
	infoConns int
	infoCmds  int
}

// NewServer makes a server listening on addr. Close with .Close().
func NewServer(addr string) (*Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return newServer(l), nil
}

func NewServerTLS(addr string, cfg *tls.Config) (*Server, error) {
	l, err := tls.Listen("tcp", addr, cfg)
	if err != nil {
		return nil, err
	}
	return newServer(l), nil
}

func newServer(l net.Listener) *Server {
	s := Server{
		cmds:     map[string]Cmd{},
		peers:    map[net.Conn]struct{}{},
		listener: l,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.serve(l)
		s.close()
	}()
	return &s
}

func (s *Server) close() {
	s.mu.Lock()
	for c := range s.peers {
		c.Close()
	}
	s.mu.Unlock()
}

func (s *Server) serve(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		s.ServeConn(conn)
	}
}

// ServeConn handles a net.Conn. Nice with net.Pipe()
func (s *Server) ServeConn(conn net.Conn) {
	s.wg.Add(1)
	s.mu.Lock()
	s.peers[conn] = struct{}{}
	s.infoConns++
	s.mu.Unlock()

	go func() {
		defer s.wg.Done()
		defer conn.Close()

		s.servePeer(conn)

		s.mu.Lock()
		delete(s.peers, conn)
		s.mu.Unlock()
	}()
}
