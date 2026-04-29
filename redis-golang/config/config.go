package config

type Config struct {
	Host string
	Port int
}

var GlobalConfig = &Config{
	Host: "0.0.0.0",
	Port: 6379,
}

func InitConfig(host string, port int) {
	GlobalConfig.Host = host
	GlobalConfig.Port = port
}
