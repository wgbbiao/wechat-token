package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/devfeel/dotweb"
)

//ResBody ResBody
type ResBody struct {
	Status      string `json:"status"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

var message = ResBody{
	Status:      "failed",
	AccessToken: "",
	ExpiresIn:   0,
}

func tokenHandler(ctx dotweb.Context) error {
	appid := ctx.QueryString("appid")
	pwd := ctx.QueryString("pwd")
	if appid == "" {
		log.Println("ERROR: 没有提供AppID参数")
		return ctx.WriteJsonC(http.StatusNotFound, message)
	}

	if secret, isExist := app.Accounts[appid]; isExist {
		//检查密码
		if pwd != app.Passwords[appid] {
			return ctx.WriteJsonC(http.StatusNotFound, message)
		}
		var accessToken string
		var recordTime string
		var expiresIn string

		// 查询数据库中是否已经存在这个AppID的access_token
		recordTime = app.Query(appid, "timestamp")
		accessToken = app.Query(appid, "access_token")
		expiresIn = app.Query(appid, "expires_in")
		expireTime, _ := strconv.ParseInt(recordTime, 10, 64)
		timeout, _ := strconv.ParseInt(expiresIn, 10, 64)

		if accessToken != "" {
			// 如果数据库中已经存在了Token，就检查过期时间，如果过期了就去GetToken获取
			curTime := time.Now().Unix()
			if curTime >= expireTime+timeout {
				token := app.WxToken.Get(appid, secret)
				// 没获得access_token就返回Failed消息
				if token == "" {
					log.Println("ERROR: 没有获得access_token.")
					return ctx.WriteJsonC(http.StatusNotFound, message)
				}

				//获取Token之后更新运行时环境，然后返回access_token
				app.UpdateToken(appid)
				message.AccessToken = app.WxToken.AccessToken
				message.ExpiresIn = int64(app.WxToken.Expire)
			} else {
				message.AccessToken = accessToken
				if app.WxToken.Expire == 0 {
					message.ExpiresIn = 7200 - (curTime - expireTime)
				} else {
					message.ExpiresIn = int64(app.WxToken.Expire) - (curTime - expireTime)
				}
			}
		} else {
			token := app.WxToken.Get(appid, secret)
			if token == "" {
				log.Println("ERROR: 没有获得access_token.")
				return ctx.WriteJsonC(http.StatusNotFound, message)
			}
			app.UpdateToken(appid)
			message.AccessToken = app.WxToken.AccessToken
			message.ExpiresIn = int64(app.WxToken.Expire)
		}

		message.Status = "success"
		return ctx.WriteJson(message)
	}

	log.Println("ERROR: AppID不存在")
	// 如果提交的appid不在配置文件中，就返回Failed消息
	return ctx.WriteJsonC(http.StatusNotFound, message)
}

//InitRoute InitRoute
func InitRoute(server *dotweb.HttpServer) {
	server.GET("/token", tokenHandler)
}
