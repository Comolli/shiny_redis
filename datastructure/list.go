package datastructure

import (
	"shiny_redis/server"
	"strconv"
	"strings"
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
	m.srv.Register("BLPOP", m.cmdBlpop)
	m.srv.Register("BRPOP", m.cmdBrpop)
	m.srv.Register("BRPOPLPUSH", m.cmdBrpoplpush)
	m.srv.Register("LINDEX", m.cmdLindex)
	m.srv.Register("LINSERT", m.cmdLinsert)
	m.srv.Register("LLEN", m.cmdLlen)
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

func (m *ShinyRedis) cmdBrpop(c *server.Peer, cmd string, args []string) {
	m.cmdBXpop(c, cmd, args, right)
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

func (m *ShinyRedis) cmdBrpoplpush(c *server.Peer, cmd string, args []string) {
	if len(args) != 3 {
		//setDirty(c)
		c.WriteError("errWrongNumber(cmd)")
		return
	}
	//todo
	//handleAuth
	//checkpub

	src := args[0]
	dst := args[1]

	timeout, err := strconv.Atoi(args[2])
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

			if !db.exists(src) {
				return false
			}
			if db.t(src) != "list" || (db.exists(dst) && db.t(dst) != "list") {
				c.WriteError("msgWrongType")
				return true
			}
			if len(db.listKeys[src]) == 0 {
				return false
			}
			elem := db.listPop(src)
			db.listLpush(dst, elem)
			c.WriteBulk(elem)
			return true
		},
		func(c *server.Peer) {
			// timeout
			c.WriteBulk("timeout")
		},
	)
}

// LINDEX
func (m *ShinyRedis) cmdLindex(c *server.Peer, cmd string, args []string) {
	if len(args) != 2 {
		//	setDirty(c)
		c.WriteError("errWrongNumber")
		return
	}
	//handleAuth
	//checkpub

	key, offsets := args[0], args[1]

	offset, err := strconv.Atoi(offsets)
	if err != nil || offsets == "-0" {
		//setDirty(c)
		c.WriteError("msgInvalidInt")
		return
	}

	withTx(m, c, func(c *server.Peer, ctx *connCtx) {
		db := m.db(ctx.selectedDB)

		t, ok := db.keys[key]
		if !ok {
			// No such key
			//c.WriteNull()
			return
		}
		if t != "list" {
			c.WriteError("msgWrongType")
			return
		}

		l := db.listKeys[key]
		if offset < 0 {
			offset = len(l) + offset
		}
		if offset < 0 || offset > len(l)-1 {
			//c.WriteNull()
			return
		}
		c.WriteBulk(l[offset])
	})
}

// LINSERT
func (m *ShinyRedis) cmdLinsert(c *server.Peer, cmd string, args []string) {
	if len(args) != 4 {
		//setDirty(c)
		c.WriteError("errWrongNumber(cmd)")
		return
	}
	//handleAuth
	//checkpub

	key := args[0]
	where := 0
	switch strings.ToLower(args[1]) {
	case "before":
		where = -1
	case "after":
		where = +1
	default:
		//setDirty(c)
		c.WriteError("msgSyntaxError")
		return
	}
	pivot := args[2]
	value := args[3]

	withTx(m, c, func(c *server.Peer, ctx *connCtx) {
		db := m.db(ctx.selectedDB)

		t, ok := db.keys[key]
		if !ok {
			// No such key
			///c.WriteInt(0)
			return
		}
		if t != "list" {
			c.WriteError("msgWrongType")
			return
		}

		l := db.listKeys[key]
		for i, el := range l {
			if el != pivot {
				continue
			}

			if where < 0 {
				l = append(l[:i], append(listKey{value}, l[i:]...)...)
			} else {
				if i == len(l)-1 {
					l = append(l, value)
				} else {
					l = append(l[:i+1], append(listKey{value}, l[i+1:]...)...)
				}
			}
			db.listKeys[key] = l
			db.keyVersion[key]++
			//c.WriteInt(len(l))
			return
		}
		//c.WriteInt(-1)
	})
}

// LLEN
func (m *ShinyRedis) cmdLlen(c *server.Peer, cmd string, args []string) {
	if len(args) != 1 {
		//setDirty(c)
		c.WriteError("errWrongNumber(cmd)")
		return
	}
	//handleAuth
	//checkpub
	key := args[0]

	withTx(m, c, func(c *server.Peer, ctx *connCtx) {
		db := m.db(ctx.selectedDB)

		t, ok := db.keys[key]
		if !ok {
			// No such key. That's zero length.
			//c.WriteInt(0)
			return
		}
		if t != "list" {
			c.WriteError("msgWrongType")
			return
		}

		c.WriteInt(len(db.listKeys[key]))
	})
}
