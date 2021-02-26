package datastructure

import (
	"regexp"
	"sync"
)

type PubsubMessage struct {
	Channel string
	Message string
}

type PubsubPmessage struct {
	Pattern string
	Channel string
	Message string
}

// Subscriber has the (p)subscriptions.
type Subscriber struct {
	publish  chan PubsubMessage
	ppublish chan PubsubPmessage
	channels map[string]struct{}
	patterns map[string]*regexp.Regexp
	mu       sync.Mutex
}
