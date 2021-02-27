package datastructure

import (
	"shiny_redis/server"
	"time"
)

type hashKey map[string]string
type listKey []string
type setKey map[string]struct{}

type RedisDB struct {
	master     *ShinyRedis              // pointer to the lock in Miniredis
	id         int                      // db id
	keys       map[string]string        // Master map of keys with their type
	stringKeys map[string]string        // GET/SET &c. keys
	hashKeys   map[string]hashKey       // MGET/MSET &c. keys
	listKeys   map[string]listKey       // LPUSH &c. keys
	setKeys    map[string]setKey        // SADD &c. keys
	ttl        map[string]time.Duration // effective TTL values
	keyVersion map[string]uint          // used to watch values
}

type dbKey struct {
	db  int
	key string
}

type txCmd func(*server.Peer, *connCtx)

// connCtx has all state for a single connection.
type connCtx struct {
	selectedDB       int            // selected DB
	authenticated    bool           // auth enabled and a valid AUTH seen
	transaction      []txCmd        // transaction callbacks. Or nil.
	dirtyTransaction bool           // any error during QUEUEing
	watch            map[dbKey]uint // WATCHed keys
	subscriber       *Subscriber    // client is in PUBSUB mode if not nil
	nested           bool           // this is called via Lua
}

// get DB. No locks!
func (m *ShinyRedis) db(i int) *RedisDB {
	if db, ok := m.Dbs[i]; ok {
		return db
	}
	db := newRedisDB(i, m) // main miniredis has our mutex.
	m.Dbs[i] = &db
	return &db
}

func newRedisDB(id int, m *ShinyRedis) RedisDB {
	return RedisDB{
		id:         id,
		master:     m,
		keys:       map[string]string{},
		stringKeys: map[string]string{},
		hashKeys:   map[string]hashKey{},
		listKeys:   map[string]listKey{},
		setKeys:    map[string]setKey{},
		ttl:        map[string]time.Duration{},
		keyVersion: map[string]uint{},
	}
}

func (db *RedisDB) exists(k string) bool {
	_, ok := db.keys[k]
	return ok
}

// t gives the type of a key,
func (db *RedisDB) t(k string) string {
	return db.keys[k]
}

// blockCmd is executed returns whether it is done
type blockCmd func(*server.Peer, *connCtx) bool

func blocking(
	m *ShinyRedis,
	c *server.Peer,
	timeout time.Duration,
	fn blockCmd,
	onTimeout func(*server.Peer),
) {

	ctx := getCtx(c)
	if inTx(ctx) {
		addTxCmd(ctx, func(c *server.Peer, ctx *connCtx) {
			if !fn(c, ctx) {
				onTimeout(c)
			}
			c.WriteInline("QUEUED")
			return
		})
	}

}

func addTxCmd(ctx *connCtx, cb txCmd) {
	ctx.transaction = append(ctx.transaction, cb)
}

func inTx(ctx *connCtx) bool {
	return ctx.transaction != nil
}

func getCtx(c *server.Peer) *connCtx {
	if c.Ctx == nil {
		c.Ctx = &connCtx{}
	}
	return c.Ctx.(*connCtx)
}

// 'left pop', aka shift.
func (db *RedisDB) listLpop(k string) string {
	l := db.listKeys[k]
	el := l[0]
	l = l[1:]
	if len(l) == 0 {
		db.del(k, true)
	} else {
		db.listKeys[k] = l
	}
	db.keyVersion[k]++
	return el
}

func (db *RedisDB) listPop(k string) string {
	l := db.listKeys[k]
	el := l[len(l)-1]
	l = l[:len(l)-1]
	if len(l) == 0 {
		db.del(k, true)
	} else {
		db.listKeys[k] = l
		db.keyVersion[k]++
	}
	return el
}

func (db *RedisDB) del(k string, delTTL bool) {
	if !db.exists(k) {
		return
	}
	t := db.t(k)
	delete(db.keys, k)
	db.keyVersion[k]++
	if delTTL {
		delete(db.ttl, k)
	}
	switch t {
	case "string":
		delete(db.stringKeys, k)
	case "hash":
		delete(db.hashKeys, k)
	case "list":
		delete(db.listKeys, k)
	case "set":
		delete(db.setKeys, k)
	case "zset":
		//delete(db.sortedsetKeys, k)
	case "stream":
		//delete(db.streamKeys, k)
	default:
		panic("Unknown key type: " + t)
	}
}
