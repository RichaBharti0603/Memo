package core

import (
	"errors"
	"io"
	"strconv"
	"time"

	"redis_golang/internal/metrics"
	"redis_golang/internal/protocol/resp"
	"redis_golang/internal/storage/memory"
	"redis_golang/internal/storage/persistence"
)

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

func EvalAndRespond(cmd *RedisCmd, c io.ReadWriter) error {
	metrics.IncCmd()
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
	default:
		return evalPING(cmd.Args, c)
	}
}
