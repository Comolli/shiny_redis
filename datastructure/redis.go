package datastructure

import (
	"context"
	"math/rand"
	"shiny_redis/server"
	"sync"
	"time"
)

type ShinyRedis struct {
	sync.Mutex
	Srv         *server.Server
	Port        int
	Passwords   map[string]string // username password
	Dbs         map[int]*RedisDB
	Scripts     map[string]string // sha1 -> lua src
	Signal      *sync.Cond
	Now         time.Time // time.Now() if not set.
	Subscribers map[*Subscriber]struct{}
	Rand        *rand.Rand
	Ctx         context.Context
	CtxCancel   context.CancelFunc
}
