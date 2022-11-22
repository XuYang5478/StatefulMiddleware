package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var SySFuncName = "sys_image_sync"

func init() {
	logFile, err := os.OpenFile("./logs.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("无法打开日志文件:", err)
		return
	}
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	// 开启 Gossip 同步
	nodeInfo, err := StartGossip(SySFuncName)
	if err != nil {
		log.Printf("[error] 无法开启 Gossip 同步: %v\n", err)
	}

	// 启动 http 服务器
	http.HandleFunc("/image/check", CheckImageHandler)
	http.HandleFunc("/container/commit", CommitImageHandler)
	http.HandleFunc("/container/start", StartContainerHandler)
	http.HandleFunc("/container/stop", StopContainerHandler)
	go func() {
		err2 := http.ListenAndServe(":8000", nil)
		if err2 != nil {
			log.Fatalln("[error], 无法启动服务器。", err2.Error())
		}
		log.Println("[info] 开启 HTTP 服务")
	}()

	quit := make(chan os.Signal, 1)
	defer close(quit)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	q := <-quit // 正常退出
	fmt.Printf("获取到信号%s\n", q)
	log.Printf("获取到信号%s\n", q)
	DeleteNode(nodeInfo)
}
