package datastructure

import "shiny_redis/server"

// commandsTransaction handles MULTI &c.
func commandsTransaction(m *ShinyRedis) {
	m.srv.Register("DISCARD", m.cmdDiscard)
	m.srv.Register("EXEC", m.cmdExec)
	m.srv.Register("MULTI", m.cmdMulti)
	m.srv.Register("UNWATCH", m.cmdUnwatch)
	//m.srv.Register("WATCH", m.cmdWatch)
}

// DISCARD
func (m *ShinyRedis) cmdDiscard(c *server.Peer, cmd string, args []string) {
	if len(args) != 0 {
		//setDirty(c)
		c.WriteError("errWrongNumber(cmd)")
		return
	}
	//handleAuth
	//checkPusbsub
	ctx := getCtx(c)
	if !inTx(ctx) {
		c.WriteError("ERR DISCARD without MULTI")
		return
	}

	stopTx(ctx)
	//c.WriteOK()
}
func inTx(ctx *connCtx) bool {
	return ctx.transaction != nil
}
func startTx(ctx *connCtx) {
	ctx.transaction = []txCmd{}
	ctx.dirtyTransaction = false
}

func stopTx(ctx *connCtx) {
	ctx.transaction = nil
	unwatch(ctx)
}

func unwatch(ctx *connCtx) {
	ctx.watch = nil
}

func watch(db *RedisDB, ctx *connCtx, key string) {
	if ctx.watch == nil {
		ctx.watch = map[dbKey]uint{}
	}
	ctx.watch[dbKey{db: db.id, key: key}] = db.keyVersion[key] // Can be 0.
}

// EXEC
func (m *ShinyRedis) cmdExec(c *server.Peer, cmd string, args []string) {
	if len(args) != 0 {
		//setDirty(c)
		c.WriteError("errWrongNumber(cmd)")
		return
	}
	//handleAuth
	//checkPusbsub

	ctx := getCtx(c)
	if ctx.nested {
		c.WriteError("msgNotFromScripts")
		return
	}
	if !inTx(ctx) {
		c.WriteError("ERR EXEC without MULTI")
		return
	}

	if ctx.dirtyTransaction {
		c.WriteError("EXECABORT Transaction discarded because of previous errors.")
		// a failed EXEC finishes the tx
		stopTx(ctx)
		return
	}

	m.Lock()
	defer m.Unlock()

	// Check WATCHed keys.
	for t, version := range ctx.watch {
		if m.db(t.db).keyVersion[t.key] > version {
			// Abort! Abort!
			stopTx(ctx)
			//c.WriteNull()
			return
		}
	}

	c.WriteLen(len(ctx.transaction))
	for _, fn := range ctx.transaction {
		fn(c, ctx)
	}
	// wake up anyone who waits on anything.
	m.signal.Broadcast()

	stopTx(ctx)
}

// MULTI
func (m *ShinyRedis) cmdMulti(c *server.Peer, cmd string, args []string) {
	if len(args) != 0 {
		c.WriteError("errWrongNumber(cmd)")
		return
	}
	//handleAuth
	//checkPusbsub

	ctx := getCtx(c)
	if ctx.nested {
		c.WriteError("msgNotFromScripts")
		return
	}
	if inTx(ctx) {
		c.WriteError("ERR MULTI calls can not be nested")
		return
	}

	startTx(ctx)

	//c.WriteOK()
}

// UNWATCH
func (m *ShinyRedis) cmdUnwatch(c *server.Peer, cmd string, args []string) {
	if len(args) != 0 {
		//setDirty(c)
		c.WriteError("errWrongNumber(cmd)")
		return
	}
	//handleAuth
	//checkPusbsub

	// Doesn't matter if UNWATCH is in a TX or not. Looks like a Redis bug to me.
	unwatch(getCtx(c))

	withTx(m, c, func(c *server.Peer, ctx *connCtx) {
		// Do nothing if it's called in a transaction.
		//c.WriteOK()
	})
}

// WATCH
func (m *ShinyRedis) cmdWatch(c *server.Peer, cmd string, args []string) {
	if len(args) == 0 {
		//setDirty(c)
		c.WriteError("errWrongNumber(cmd)")
		return
	}
	//handleAuth
	//checkPusbsub

	ctx := getCtx(c)
	if ctx.nested {
		c.WriteError("msgNotFromScripts")
		return
	}
	if inTx(ctx) {
		c.WriteError("ERR WATCH in MULTI")
		return
	}

	m.Lock()
	defer m.Unlock()
	db := m.db(ctx.selectedDB)

	for _, key := range args {
		watch(db, ctx, key)
	}
	//c.WriteOK()
}
