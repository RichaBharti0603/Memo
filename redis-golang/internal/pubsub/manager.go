package pubsub

import (
	"io"
	"sync"
	"redis_golang/internal/metrics"
	"redis_golang/internal/protocol/resp"
)

var (
	// channels maps channel name -> list of subscriber connections
	channels = make(map[string][]io.ReadWriter)
	mu       sync.RWMutex
)

// Subscribe adds a connection to the specified channels
func Subscribe(c io.ReadWriter, chs []string) int {
	mu.Lock()
	defer mu.Unlock()

	for _, ch := range chs {
		channels[ch] = append(channels[ch], c)
	}
	metrics.SetActiveChannels(int64(len(channels)))
	return len(chs)
}

// Unsubscribe removes a connection from all channels or specified ones
func Unsubscribe(c io.ReadWriter, chs []string) {
	mu.Lock()
	defer mu.Unlock()

	if len(chs) == 0 {
		// Unsubscribe from all
		for ch, subs := range channels {
			var newSubs []io.ReadWriter
			for _, sub := range subs {
				if sub != c {
					newSubs = append(newSubs, sub)
				}
			}
			if len(newSubs) == 0 {
				delete(channels, ch)
			} else {
				channels[ch] = newSubs
			}
		}
	} else {
		for _, ch := range chs {
			subs := channels[ch]
			var newSubs []io.ReadWriter
			for _, sub := range subs {
				if sub != c {
					newSubs = append(newSubs, sub)
				}
			}
			if len(newSubs) == 0 {
				delete(channels, ch)
			} else {
				channels[ch] = newSubs
			}
		}
	}
	metrics.SetActiveChannels(int64(len(channels)))
}

// Publish sends a message to all subscribers of a channel and returns the number of recipients
func Publish(ch string, msg string) int {
	mu.Lock()
	defer mu.Unlock()

	subs, exists := channels[ch]
	if !exists {
		return 0
	}

	payload := resp.EncodeArray([]string{"message", ch, msg})
	
	var activeSubs []io.ReadWriter
	count := 0
	for _, sub := range subs {
		_, err := sub.Write(payload)
		if err == nil {
			activeSubs = append(activeSubs, sub)
			count++
		}
	}

	if len(activeSubs) == 0 {
		delete(channels, ch)
	} else {
		channels[ch] = activeSubs
	}

	metrics.SetActiveChannels(int64(len(channels)))
	metrics.IncPubSubMsg()
	return count
}
