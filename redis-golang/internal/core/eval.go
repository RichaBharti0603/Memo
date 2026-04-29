package core

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"redis_golang/internal/metrics"
	"redis_golang/internal/protocol/resp"
	"redis_golang/internal/pubsub"
	"redis_golang/internal/replication"
	"redis_golang/internal/storage/memory"
	"redis_golang/internal/storage/persistence"
)

func EvalAndRespond(command string, args []string, c io.ReadWriter) error {
	switch strings.ToUpper(command) {
	case "PING":
		return evalPING(args, c)
	case "SET":
		return evalSET(args, c)
	case "GET":
		return evalGET(args, c)
	case "TTL":
		return evalTTL(args, c)
	case "DEL":
		return evalDEL(args, c)
	case "INCR":
		return evalINCR(args, c)
	case "DECR":
		return evalDECR(args, c)
	case "SUBSCRIBE":
		return evalSUBSCRIBE(args, c)
	case "UNSUBSCRIBE":
		return evalUNSUBSCRIBE(args, c)
	case "PUBLISH":
		return evalPUBLISH(args, c)
	default:
		return errors.New("ERR unknown command")
	}
}

func evalPING(args []string, c io.ReadWriter) error {
	var b []byte

	if len(args) >= 2 {
		return errors.New("ERR wrong number of arguments for 'ping' command")
	}

	if len(args) == 0 {
		b = resp.Encode("PONG", true)
	} else {
		b = resp.Encode(args[0], false)
	}

	_, err := c.Write(b)
	return err
}

func evalSET(args []string, c io.ReadWriter) error {
	if len(args) <= 1 {
		return errors.New("ERR wrong number of arguments for 'set' command")
	}

	var key, value string
	var exDurationMs int64 = -1

	key, value = args[0], args[1]

	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "EX", "ex":
			i++
			if i == len(args) {
				return errors.New("ERR syntax error")
			}

			exDurationSec, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return errors.New("ERR value is not an integer or out of range")
			}
			exDurationMs = exDurationSec * 1000
		default:
			return errors.New("ERR syntax error")
		}
	}

	memory.Put(key, memory.NewObj(value, exDurationMs))
	
	if persistence.GlobalAOF != nil {
		persistence.GlobalAOF.WriteCmd("SET", args)
	}

	if replication.GlobalRole == replication.RolePrimary {
		replication.Broadcast("SET", args)
	}

	c.Write([]byte("+OK\r\n"))
	return nil
}

func evalGET(args []string, c io.ReadWriter) error {
	if len(args) != 1 {
		return errors.New("ERR wrong number of arguments for 'get' command")
	}

	var key string = args[0]
	obj := memory.Get(key)

	if obj == nil {
		metrics.IncMiss()
		c.Write(resp.Nil)
		return nil
	}

	if obj.ExpiresAt != -1 && obj.ExpiresAt <= time.Now().UnixMilli() {
		metrics.IncMiss()
		c.Write(resp.Nil)
		return nil
	}

	metrics.IncHit()
	c.Write(resp.Encode(obj.Value, false))
	return nil
}

func evalTTL(args []string, c io.ReadWriter) error {
	if len(args) != 1 {
		return errors.New("ERR wrong number of arguments for 'ttl' command")
	}

	var key string = args[0]
	obj := memory.Get(key)

	if obj == nil {
		c.Write([]byte(":-2\r\n"))
		return nil
	}

	if obj.ExpiresAt == -1 {
		c.Write([]byte(":-1\r\n"))
		return nil
	}

	durationMs := obj.ExpiresAt - time.Now().UnixMilli()

	if durationMs < 0 {
		c.Write([]byte(":-2\r\n"))
		return nil
	}

	c.Write(resp.Encode(int64(durationMs/1000), false))
	return nil
}

func evalDEL(args []string, c io.ReadWriter) error {
	if len(args) == 0 {
		return errors.New("ERR wrong number of arguments for 'del' command")
	}

	var countDeleted int64 = 0

	for _, key := range args {
		if memory.Del(key) {
			countDeleted++
			if persistence.GlobalAOF != nil {
				persistence.GlobalAOF.WriteCmd("DEL", []string{key})
			}
			if replication.GlobalRole == replication.RolePrimary {
				replication.Broadcast("DEL", []string{key})
			}
		}
	}

	c.Write(resp.Encode(countDeleted, false))
	return nil
}

func evalINCR(args []string, c io.ReadWriter) error {
	if len(args) != 1 {
		return errors.New("ERR wrong number of arguments for 'incr' command")
	}
	key := args[0]
	obj := memory.Get(key)
	var val int64 = 0
	if obj != nil {
		if obj.ExpiresAt != -1 && obj.ExpiresAt <= time.Now().UnixMilli() {
			val = 0
		} else {
			strVal, ok := obj.Value.(string)
			if !ok {
				return errors.New("ERR value is not an integer or out of range")
			}
			parsed, err := strconv.ParseInt(strVal, 10, 64)
			if err != nil {
				return errors.New("ERR value is not an integer or out of range")
			}
			val = parsed
		}
	}
	val++
	memory.Put(key, memory.NewObj(strconv.FormatInt(val, 10), -1))
	if persistence.GlobalAOF != nil {
		persistence.GlobalAOF.WriteCmd("INCR", args)
	}
	if replication.GlobalRole == replication.RolePrimary {
		replication.Broadcast("INCR", args)
	}
	c.Write(resp.Encode(val, false))
	return nil
}

func evalDECR(args []string, c io.ReadWriter) error {
	if len(args) != 1 {
		return errors.New("ERR wrong number of arguments for 'decr' command")
	}
	key := args[0]
	obj := memory.Get(key)
	var val int64 = 0
	if obj != nil {
		if obj.ExpiresAt != -1 && obj.ExpiresAt <= time.Now().UnixMilli() {
			val = 0
		} else {
			strVal, ok := obj.Value.(string)
			if !ok {
				return errors.New("ERR value is not an integer or out of range")
			}
			parsed, err := strconv.ParseInt(strVal, 10, 64)
			if err != nil {
				return errors.New("ERR value is not an integer or out of range")
			}
			val = parsed
		}
	}
	val--
	memory.Put(key, memory.NewObj(strconv.FormatInt(val, 10), -1))
	if persistence.GlobalAOF != nil {
		persistence.GlobalAOF.WriteCmd("DECR", args)
	}
	if replication.GlobalRole == replication.RolePrimary {
		replication.Broadcast("DECR", args)
	}
	c.Write(resp.Encode(val, false))
	return nil
}

func evalHSET(args []string, c io.ReadWriter) error {
	if len(args) != 3 {
		return errors.New("ERR wrong number of arguments for 'hset' command")
	}
	key, field, value := args[0], args[1], args[2]

	obj := memory.Get(key)
	var hash map[string]string
	var ttl int64 = -1

	if obj == nil {
		hash = make(map[string]string)
	} else {
		if obj.ExpiresAt != -1 && obj.ExpiresAt <= time.Now().UnixMilli() {
			hash = make(map[string]string)
		} else {
			var ok bool
			hash, ok = obj.Value.(map[string]string)
			if !ok {
				return errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
			}
			if obj.ExpiresAt != -1 {
				ttl = obj.ExpiresAt - time.Now().UnixMilli()
			}
		}
	}

	hash[field] = value
	memory.Put(key, memory.NewObj(hash, ttl))

	if persistence.GlobalAOF != nil {
		persistence.GlobalAOF.WriteCmd("HSET", args)
	}
	if replication.GlobalRole == replication.RolePrimary {
		replication.Broadcast("HSET", args)
	}
	
	c.Write(resp.Encode(int64(1), false))
	return nil
}

func evalHGET(args []string, c io.ReadWriter) error {
	if len(args) != 2 {
		return errors.New("ERR wrong number of arguments for 'hget' command")
	}
	key, field := args[0], args[1]

	obj := memory.Get(key)
	if obj == nil {
		metrics.IncMiss()
		c.Write(resp.Nil)
		return nil
	}

	if obj.ExpiresAt != -1 && obj.ExpiresAt <= time.Now().UnixMilli() {
		metrics.IncMiss()
		c.Write(resp.Nil)
		return nil
	}

	hash, ok := obj.Value.(map[string]string)
	if !ok {
		return errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	val, exists := hash[field]
	if !exists {
		metrics.IncMiss()
		c.Write(resp.Nil)
		return nil
	}

	metrics.IncHit()
	c.Write(resp.Encode(val, false))
	return nil
}

func evalSUBSCRIBE(args []string, c io.ReadWriter) error {
	if len(args) == 0 {
		return errors.New("ERR wrong number of arguments for 'subscribe' command")
	}
	
	// Register subscriber
	pubsub.Subscribe(c, args)
	
	// Acknowledge subscription for each channel
	for i, ch := range args {
		payload := resp.EncodeArray([]string{"subscribe", ch, strconv.Itoa(i + 1)})
		c.Write(payload)
	}
	return nil
}

func evalUNSUBSCRIBE(args []string, c io.ReadWriter) error {
	pubsub.Unsubscribe(c, args)
	
	if len(args) == 0 {
		// Redis behavior: returns unsubscribe payload for all channels.
		// For MVP, we just send a generic OK or let the client close.
		c.Write(resp.EncodeArray([]string{"unsubscribe", "", "0"}))
	} else {
		for i, ch := range args {
			payload := resp.EncodeArray([]string{"unsubscribe", ch, strconv.Itoa(len(args) - (i + 1))})
			c.Write(payload)
		}
	}
	return nil
}

func evalPUBLISH(args []string, c io.ReadWriter) error {
	if len(args) != 2 {
		return errors.New("ERR wrong number of arguments for 'publish' command")
	}
	channel, message := args[0], args[1]
	
	receivers := pubsub.Publish(channel, message)
	
	if replication.GlobalRole == replication.RolePrimary {
		replication.Broadcast("PUBLISH", args)
	}
	
	c.Write(resp.Encode(int64(receivers), false))
	return nil
}

func isWriteCommand(cmd string) bool {
	switch cmd {
	case "SET", "DEL", "INCR", "DECR", "HSET":
		return true
	}
	return false
}

func EvalAndRespond(cmd *RedisCmd, c io.ReadWriter) error {
	metrics.IncCmd()

	if isWriteCommand(cmd.Cmd) && replication.GlobalRole == replication.RoleReplica {
		c.Write([]byte("-READONLY You can't write against a read only replica.\r\n"))
		return nil
	}

	return EvalCommandUnsafe(cmd, c)
}

func evalSYNC(args []string, c io.ReadWriter) error {
	// Only master should handle SYNC
	if replication.GlobalRole != replication.RolePrimary {
		c.Write([]byte("-ERR I am not a master\r\n"))
		return nil
	}
	replication.HandleSync(c)
	return nil
}

func evalREPLICAOF(args []string, c io.ReadWriter) error {
	if len(args) != 2 {
		return errors.New("ERR wrong number of arguments for 'replicaof' command")
	}
	host := args[0]
	portStr := args[1]
	
	if host == "NO" && portStr == "ONE" {
		replication.GlobalRole = replication.RolePrimary
		c.Write([]byte("+OK\r\n"))
		return nil
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return errors.New("ERR invalid port")
	}

	// Start replica in background
	go replication.StartReplica(host, port, func(cmd string, args []string, c io.ReadWriter) error {
		return EvalCommandUnsafe(&RedisCmd{Cmd: cmd, Args: args}, c)
	})

	c.Write([]byte("+OK\r\n"))
	return nil
}

func EvalCommandUnsafe(cmd *RedisCmd, c io.ReadWriter) error {
	switch cmd.Cmd {
	case "PING":
		return evalPING(cmd.Args, c)
	case "SET":
		return evalSET(cmd.Args, c)
	case "GET":
		return evalGET(cmd.Args, c)
	case "TTL":
		return evalTTL(cmd.Args, c)
	case "DEL":
		return evalDEL(cmd.Args, c)
	case "INCR":
		return evalINCR(cmd.Args, c)
	case "DECR":
		return evalDECR(cmd.Args, c)
	case "HSET":
		return evalHSET(cmd.Args, c)
	case "HGET":
		return evalHGET(cmd.Args, c)
	case "SYNC":
		return evalSYNC(cmd.Args, c)
	case "REPLICAOF":
		return evalREPLICAOF(cmd.Args, c)
	case "SUBSCRIBE":
		return evalSUBSCRIBE(cmd.Args, c)
	case "UNSUBSCRIBE":
		return evalUNSUBSCRIBE(cmd.Args, c)
	case "PUBLISH":
		return evalPUBLISH(cmd.Args, c)
	default:
		return evalPING(cmd.Args, c)
	}
}
