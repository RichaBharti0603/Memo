package metrics

import "sync/atomic"

var (
	ActiveConnections      atomic.Int64
	TotalCommandsProcessed atomic.Int64
	CacheHits              atomic.Int64
	CacheMisses            atomic.Int64
	ConnectedReplicas      atomic.Int64
	ActiveChannels         atomic.Int64
	PubSubMessages         atomic.Int64
)

// Incrementers
func IncConn() { ActiveConnections.Add(1) }
func DecConn() { ActiveConnections.Add(-1) }
func IncCmd()  { TotalCommandsProcessed.Add(1) }
func IncHit()  { CacheHits.Add(1) }
func IncMiss() { CacheMisses.Add(1) }
func IncReplica() { ConnectedReplicas.Add(1) }
func DecReplica() { ConnectedReplicas.Add(-1) }
func SetActiveChannels(val int64) { ActiveChannels.Store(val) }
func IncPubSubMsg() { PubSubMessages.Add(1) }

// Getters
func GetActiveConnections() int64 { return ActiveConnections.Load() }
func GetTotalCommands() int64     { return TotalCommandsProcessed.Load() }
func GetCacheHits() int64         { return CacheHits.Load() }
func GetCacheMisses() int64       { return CacheMisses.Load() }
func GetConnectedReplicas() int64 { return ConnectedReplicas.Load() }
func GetActiveChannels() int64    { return ActiveChannels.Load() }
func GetPubSubMessages() int64    { return PubSubMessages.Load() }
