package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/devfeel/dotweb"
)

//App App
type App struct {
	Accounts  map[string]string
	Passwords map[string]string
	RC        redis.Conn
	Web       *dotweb.DotWeb
	WxToken   *Token
}

//Account Account
type Account struct {
	AppID    string `json:"appid"`
	Secret   string `json:"secret"`
	Password string `json:"password"`
}

//NewApp NewApp
func NewApp() *App {
	var a = &App{}
	a.Accounts = make(map[string]string)
	a.Passwords = make(map[string]string)
	a.Web = dotweb.New()
	a.WxToken = new(Token)

	return a
}

//SetAccounts 读取配置文件中的appid和secret值到一个map中
func (a *App) SetAccounts(config *string) {
	accounts := make([]Account, 1)

	if _, err := os.Stat(*config); err != nil {
		fmt.Println("配置文件无法打开！")
		os.Exit(1)
	}

	raw, err := ioutil.ReadFile(*config)
	if err != nil {
		fmt.Println("配置文件读取失败！")
		os.Exit(1)
	}

	if err := json.Unmarshal(raw, &accounts); err != nil {
		fmt.Println("配置文件内容错误！")
		os.Exit(1)
	}

	for _, acc := range accounts {
		a.Accounts[acc.AppID] = acc.Secret
		a.Passwords[acc.AppID] = acc.Password
	}
}

//Query Query
func (a *App) Query(appid string, key string) string {
	if r, err := redis.String(a.RC.Do("GET", appid+"_"+key)); r != "" && err == nil {
		return r
	}
	return ""
}

//UpdateToken 更新AppID上下文环境中的Access Token和到期时间
func (a *App) UpdateToken(appid string) {
	timestamp := time.Now().Unix()
	a.RC.Do("SET", appid+"_timestamp", strconv.FormatInt(timestamp, 10))
	a.RC.Do("SET", appid+"_access_token", a.WxToken.AccessToken)
	a.RC.Do("SET", appid+"_expires_in", strconv.Itoa(a.WxToken.Expire))
}

//UpdateTokenDaemon 后台刷新token
func (a *App) UpdateTokenDaemon(appid, secret string, initTickDuration time.Duration) {
	fmt.Println("UpdateTokenDaemon:", time.Now())
	fmt.Println("initTickDuration:", initTickDuration)
	tickDuration := initTickDuration
	ticker := time.NewTicker(tickDuration)
	select {
	case <-ticker.C:
		token := a.WxToken.Get(appid, secret)
		// 没获得access_token就返回Failed消息
		if token == "" {
			log.Println("ERROR: 没有获得access_token.")
		}

		//获取Token之后更新运行时环境，然后返回access_token
		a.UpdateToken(appid)
		ticker.Stop()
	}
}

//StartUpdateToken 启动时制定定时刷新计划
func (a *App) StartUpdateToken() {
	for appid, secret := range a.Accounts {
		curTime := time.Now().Unix()
		expiresIn := app.Query(appid, "expires_in")
		recordTime := app.Query(appid, "timestamp")
		timeout, _ := strconv.ParseInt(expiresIn, 10, 64)
		expireTime, _ := strconv.ParseInt(recordTime, 10, 64)
		sce := timeout - (curTime - expireTime) - 100 //提前100秒刷新
		fmt.Println("sce:", sce)
		if sce < 0 {
			sce = 1
		}
		go app.UpdateTokenDaemon(appid, secret, time.Duration(sce)*time.Second)
	}
}
