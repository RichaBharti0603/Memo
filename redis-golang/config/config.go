package config

type Config struct {
	Host        string
	Port        int
	ReplicaHost string
	ReplicaPort int
}

var GlobalConfig = &Config{
	Host: "0.0.0.0",
	Port: 6379,
}

func InitConfig(host string, port int, replicaHost string, replicaPort int) {
	GlobalConfig.Host = host
	GlobalConfig.Port = port
	GlobalConfig.ReplicaHost = replicaHost
	GlobalConfig.ReplicaPort = replicaPort
}
