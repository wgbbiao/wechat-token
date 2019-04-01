package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/go-ini/ini"
)

var app = NewApp()

func main() {

	var (
		version     = flag.Bool("version", false, "version v0.1")
		config      = flag.String("config", "account.json", "config file.")
		port        = flag.Int("port", 8000, "listen port.")
		cfg         *ini.File
		RedisClient *redis.Pool
	)

	cfg, _ = ini.LooseLoad("./config.ini")

	flag.Parse()

	if *version {
		fmt.Println("v0.1")
		os.Exit(0)
	}

	fmt.Println(cfg.Section("redis").Key("host").String())
	RedisClient = &redis.Pool{
		MaxIdle:     cfg.Section("redis").Key("maxIdle").MustInt(30),
		MaxActive:   cfg.Section("redis").Key("maxActive").MustInt(30),
		IdleTimeout: 180 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", cfg.Section("redis").Key("host").String())
			if err != nil {
				return nil, err
			}
			if cfg.Section("redis").Key("password").String() != "" {
				if _, err := c.Do("AUTH", cfg.Section("redis").Key("password").String()); err != nil {
					c.Close()
					return nil, err
				}
			}
			c.Do("SELECT", cfg.Section("redis").Key("db").MustInt(3))
			return c, nil
		},
		Wait: true,
	}
	app.SetAccounts(config)
	app.RC = RedisClient.Get()
	defer app.RC.Close()

	InitRoute(app.Web.HttpServer)
	log.Println("Start AccessToken Server on ", *port)
	app.StartUpdateToken()
	app.Web.StartServer(*port)
}
