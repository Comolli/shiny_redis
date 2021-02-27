package datastructure

import (
	"shiny_redis/server"
	"strconv"
	"time"
)

type leftright int

const (
	left leftright = iota
	right
)

// commandsList handles list commands (mostly L*)
func CommandsList(m *ShinyRedis) {
	//Time complexity: O(1)
	m.Srv.Register("BLPOP", m.cmdBlpop)
	//m.srv.Register("BRPOP", m.cmdBrpop)
	//m.srv.Register("BRPOPLPUSH", m.cmdBrpoplpush)
	//m.srv.Register("LINDEX", m.cmdLindex)
	//m.srv.Register("LINSERT", m.cmdLinsert)
	//m.srv.Register("LLEN", m.cmdLlen)
	//m.srv.Register("LPOP", m.cmdLpop)
	//m.srv.Register("LPUSH", m.cmdLpush)
	//m.srv.Register("LPUSHX", m.cmdLpushx)
	//m.srv.Register("LRANGE", m.cmdLrange)
	//m.srv.Register("LREM", m.cmdLrem)
	//m.srv.Register("LSET", m.cmdLset)
	//m.srv.Register("LTRIM", m.cmdLtrim)
	//m.srv.Register("RPOP", m.cmdRpop)
	//m.srv.Register("RPOPLPUSH", m.cmdRpoplpush)
	//m.srv.Register("RPUSH", m.cmdRpush)
	//m.srv.Register("RPUSHX", m.cmdRpushx)
}
func (m *ShinyRedis) cmdBlpop(c *server.Peer, cmd string, args []string) {
	m.cmdBXpop(c, cmd, args, left)
}

func (m *ShinyRedis) cmdBXpop(c *server.Peer, cmd string, args []string, lr leftright) {
	if len(args) < 2 {
		//setDirty(c)
		c.WriteError("errWrongNumber(cmd)")
		return
	}
	//todo
	//handleAuth
	//checkpub

	timeoutS := args[len(args)-1]
	keys := args[:len(args)-1]

	timeout, err := strconv.Atoi(timeoutS)
	if err != nil {
		//setDirty(c)
		//c.WriteError(msgInvalidTimeout)
		return
	}
	if timeout < 0 {
		//setDirty(c)
		//c.WriteError(msgNegTimeout)
		return
	}

	blocking(
		m,
		c,
		time.Duration(timeout)*time.Second,
		func(c *server.Peer, ctx *connCtx) bool {
			db := m.db(ctx.selectedDB)
			for _, key := range keys {
				if !db.exists(key) {
					continue
				}
				if db.t(key) != "list" {
					c.WriteError("msgWrongType")
					return true
				}

				if len(db.listKeys[key]) == 0 {
					continue
				}
				//c.WriteLen(2)
				//c.WriteBulk(key)
				var v string
				switch lr {
				case left:
					v = db.listLpop(key)
				case right:
					v = db.listPop(key)
				}
				c.WriteBulk(v)
				return true
			}
			return false
		},
		func(c *server.Peer) {
			// timeout
			c.WriteBulk("timeout")
		},
	)
}
