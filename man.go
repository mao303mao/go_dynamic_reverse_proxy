package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	//_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/kataras/iris"
)

// 获取自定义配置
func GetOtherConfig(config *iris.Configuration, configName string) (interface{}, error) {
	otherConfig := config.GetOther()
	configValue, ok := otherConfig[configName]
	if ok {
		return configValue, nil
	}
	return nil, errors.New("没有此Other配置:" + configName)
}

// 响应Json同一结构
type ResponseJson struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

// 响应Json构造器
func NewResponseJson(code int, data interface{}, msg string) *ResponseJson {
	rjson := &ResponseJson{}
	rjson.Code = code // 0-成功，1-失败
	rjson.Data = data
	rjson.Msg = msg
	return rjson
}

// cors处理
func GetCorsHandle(corsHost string) func(iris.Context) {
	Cors := func(ctx iris.Context) {
		ctx.Header("Access-Control-Allow-Origin", corsHost)
		ctx.Header("Access-Control-Allow-Credentials", "true")
		if ctx.Request().Method == "OPTIONS" {
			ctx.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
			ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			ctx.StatusCode(204)
			return
		}
		ctx.Next()
	}
	return Cors
}

// 定义全局的反向代理配置
var RvConf = &ReverseConf{}

// demo
func SetNewTarget(revsName, newTarget string) error {
	for _, conf := range RvConf.Confs {
		if conf.Name != revsName {
			continue
		}
		conf.Target = newTarget
		return nil
	}
	return errors.New("未找到对应反向配置名称（appName）")
}

// 给Reverse下各个route设置header
func SetFciReveser(revsName, newTarget, headerName, headerValue string) error {
	for _, conf := range RvConf.Confs {
		if conf.Name != revsName {
			continue
		}
		conf.Target = newTarget
		for _, rt := range conf.Routes {
			headerIdx := -1
			for idx, hv := range rt.ReqHeaders {
				if hv.Name == headerName {
					headerIdx = idx
					break
				}
			}
			if headerIdx < 0 && headerValue != "" { // 不存在此header
				rt.ReqHeaders = append(rt.ReqHeaders, &ReqHeader{Name: headerName, Value: headerValue})
			}
			if headerIdx >= 0 && headerValue != "" { // 存在此header
				rt.ReqHeaders[headerIdx].Value = headerValue
			}
			if headerIdx >= 0 && headerValue == "" { // 存在此header，但需要删除
				rt.ReqHeaders = append(rt.ReqHeaders[:headerIdx], rt.ReqHeaders[headerIdx+1:]...)
			}
		}
		return nil
	}
	return errors.New("未找到对应反向配置名称（appName）")
}

// 给Reverse下单个route设置header
func SetNewHeader(revsName, routeName, headerName, headerValue string) error {
	for _, conf := range RvConf.Confs {
		if conf.Name != revsName {
			continue
		}
		for _, rt := range conf.Routes {
			if rt.Name != routeName {
				continue
			}
			headerIdx := -1
			for idx, hv := range rt.ReqHeaders {
				if hv.Name == headerName {
					headerIdx = idx
					break
				}
			}
			if headerIdx < 0 && headerValue != "" { // 不存在此header
				rt.ReqHeaders = append(rt.ReqHeaders, &ReqHeader{Name: headerName, Value: headerValue})
			}
			if headerIdx >= 0 && headerValue != "" { // 存在此header
				rt.ReqHeaders[headerIdx].Value = headerValue
			}
			if headerIdx >= 0 && headerValue == "" { // 存在此header，但需要删除
				rt.ReqHeaders = append(rt.ReqHeaders[:headerIdx], rt.ReqHeaders[headerIdx+1:]...)
			}
			return nil
		}
	}
	return errors.New("未找到对应反向配置名称（appName）")
}

func main() {
	// 日志写入文件
	logfile, err := os.OpenFile("logs/std.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	if err != nil {
		return
	}
	defer logfile.Close()
	log.SetOutput(logfile)
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime | log.LstdFlags)

	// 启用app服务
	app := iris.New()
	appConfig := iris.YAML("./sysConf.yml")
	corsHostI, err := GetOtherConfig(&appConfig, "CorsHost")
	if err != nil {
		panic(err)
	}
	app.Use(GetCorsHandle(corsHostI.(string)))
	serverPortI, err := GetOtherConfig(&appConfig, "ServerPort")
	var serverPort string
	if err != nil {
		serverPort = "8080"
	} else {
		serverPort = fmt.Sprintf("%d", serverPortI.(int))
	}

	// 获取转发地址及其下路由配置
	app.Get("/getReverseInfo", func(ctx iris.Context) {
		ctx.JSON(NewResponseJson(0, RvConf, ""))
	})

	// 设置新的转发地址
	app.Get("/setNewTarget/{revsName:string}", func(ctx iris.Context) {
		revsName := strings.TrimSpace(ctx.Params().Get("revsName"))
		newTarget := strings.TrimSpace(ctx.URLParam("newTarget"))
		err := SetNewTarget(revsName, newTarget)
		if err != nil {
			ctx.JSON(NewResponseJson(1, "", err.Error()))
			return
		}
		ctx.JSON(NewResponseJson(0, "", "修改目的host成功"))
	})

	// 设置FCI的转发
	app.Get("/setFciReveser/{revsName:string}", func(ctx iris.Context) {
		revsName := strings.TrimSpace(ctx.Params().Get("revsName"))
		newTarget := strings.TrimSpace(ctx.URLParam("newTarget"))
		headerName := strings.TrimSpace(ctx.URLParam("headerName"))
		headerValue := strings.TrimSpace(ctx.URLParam("headerValue"))
		err := SetFciReveser(revsName, newTarget, headerName, headerValue)
		if err != nil {
			ctx.JSON(NewResponseJson(1, "", err.Error()))
			return
		}
		ctx.JSON(NewResponseJson(0, "", "修改目的请求头成功"))
	})

	// 设置对应路由下的请求头
	app.Get("/setNewHeader/{revsName:string}/{routeName:string}", func(ctx iris.Context) {
		revsName := strings.TrimSpace(ctx.Params().Get("revsName"))
		routeName := strings.TrimSpace(ctx.Params().Get("routeName"))
		headerName := strings.TrimSpace(ctx.URLParam("headerName"))
		headerValue := strings.TrimSpace(ctx.URLParam("headerValue"))
		err := SetNewHeader(revsName, routeName, headerName, headerValue)
		if err != nil {
			ctx.JSON(NewResponseJson(1, "", err.Error()))
			return
		}
		ctx.JSON(NewResponseJson(0, "", "修改目的请求头成功"))
	})

	// 关闭服务器时的处理
	iris.RegisterOnInterrupt(func() {
		timeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		app.Shutdown(ctx)
	})
	app.Configure(iris.WithConfiguration(appConfig))
	// 启动反向代理服务
	go RunReverseProxyServ()
	app.Run(iris.Addr("0.0.0.0:" + serverPort))
}
