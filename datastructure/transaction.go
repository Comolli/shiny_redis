package datastructure

import "shiny_redis/server"

// commandsTransaction handles MULTI &c.
func commandsTransaction(m *ShinyRedis) {
	m.srv.Register("DISCARD", m.cmdDiscard)
	m.srv.Register("EXEC", m.cmdExec)
	//m.srv.Register("MULTI", m.cmdMulti)
	//m.srv.Register("UNWATCH", m.cmdUnwatch)
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

func stopTx(ctx *connCtx) {
	ctx.transaction = nil
	unwatch(ctx)
}

func unwatch(ctx *connCtx) {
	ctx.watch = nil
}
