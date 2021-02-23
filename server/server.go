package server

import (
	"bufio"
	"net"
	"sync"
)

// Hook is can be added to run before every cmd. Return true if the command is done.
type Callback func(*Peer, string, ...string) bool
type Cmd func(c *Peer, cmd string, args []string)
type Peer struct {
	w            *bufio.Writer
	closed       bool
	Resp3        bool
	Ctx          interface{} // anything goes, server won't touch this
	onDisconnect []func()    // list of callbacks
	mu           sync.Mutex  // for Block()
}
type Server struct {
	l         net.Listener
	cmds      map[string]Cmd
	preHook   Callback
	peers     map[net.Conn]struct{}
	mu        sync.Mutex
	wg        sync.WaitGroup
	infoConns int
	infoCmds  int
}
