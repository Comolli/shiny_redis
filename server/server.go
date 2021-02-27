package server

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"shiny_redis/parser"
	"strings"
	"sync"
	"unicode"
)

// Hook is can be added to run before every cmd. Return true if the command is done.
type Callback func(*Peer, string, ...string) bool
type Cmd func(c *Peer, cmd string, args []string)

//client
type Peer struct {
	writer    *bufio.Writer
	closed    bool
	Resp3     bool
	Ctx       interface{} // anything goes, server won't touch this
	DisconnCB []func()    // list of callbacks
	mu        sync.Mutex  // for Block()
}

//server
type Server struct {
	listener  net.Listener
	cmds      map[string]Cmd
	preHook   Callback
	peers     map[net.Conn]struct{}
	mu        sync.Mutex
	wg        sync.WaitGroup
	infoConns int
	CmdCnt    int
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

func (s *Server) servePeer(c net.Conn) {
	r := bufio.NewReader(c)
	peer := &Peer{
		writer: bufio.NewWriter(c),
	}
	defer func() {
		for _, f := range peer.DisconnCB {
			f()
		}
	}()

	for {
		args, err := parser.ReadArray(r)
		if err != nil {
			return
		}
		s.Dispatch(peer, args)
		peer.Flush()

		s.mu.Lock()
		closed := peer.closed
		s.mu.Unlock()
		if closed {
			c.Close()
		}
	}
}

func (s *Server) Dispatch(c *Peer, args []string) {
	cmd, args := args[0], args[1:]
	cmdUp := strings.ToUpper(cmd)
	s.mu.Lock()
	fn := s.preHook
	s.mu.Unlock()
	if fn != nil {
		if fn(c, cmdUp, args...) {
			return
		}
	}

	s.mu.Lock()
	cb, ok := s.cmds[cmdUp]
	s.mu.Unlock()
	if !ok {
		//todo
		return
	}

	s.mu.Lock()
	s.CmdCnt++
	s.mu.Unlock()
	cb(c, cmdUp, args)
}

func (c *Peer) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writer.Flush()
}

func (c *Peer) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
}

func (s *Server) TotalCommands() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.CmdCnt
}

func (s *Server) Register(cmd string, f Cmd) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cmd = strings.ToUpper(cmd)
	if _, ok := s.cmds[cmd]; ok {
		return fmt.Errorf("command already registered: %s", cmd)
	}
	s.cmds[cmd] = f
	return nil
}

// A Writer is given to the callback in Block()
type Writer struct {
	w     *bufio.Writer
	resp3 bool
}

func (c *Peer) WriteInline(s string) {
	c.Block(func(w *Writer) {
		w.WriteInline(s)
	})
}

func (c *Peer) Block(fn func(*Writer)) {
	fn(&Writer{c.writer, c.Resp3})
}

// WriteInline writes a redis inline string
func (w *Writer) WriteInline(s string) {
	fmt.Fprintf(w.w, "+%s\r\n", toInline(s))
}

//formattting string
func toInline(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, s)
}

// WriteError writes a redis 'Error'
func (c *Peer) WriteError(e string) {
	c.Block(func(w *Writer) {
		w.WriteError(e)
	})
}

func (w *Writer) WriteError(e string) {
	fmt.Fprintf(w.w, "-%s\r\n", toInline(e))
}

// WriteBulk writes a bulk string
func (c *Peer) WriteBulk(s string) {
	c.Block(func(w *Writer) {
		w.WriteBulk(s)
	})
}

// WriteBulk writes a bulk string
func (w *Writer) WriteBulk(s string) {
	fmt.Fprintf(w.w, "$%d\r\n%s\r\n", len(s), s)
}
