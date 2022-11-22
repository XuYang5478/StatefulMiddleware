package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var UpperDir = ""

func main() {
	// 从命令行参数获取容器id
	functionName := os.Args[1]
	containerId := os.Args[2]

	// 获取容器的上层目录
	UpperDir = getUpperDir(containerId)
	// /var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/98/fs

	// w := FileWatcher("/root/gossip-demo1", nil)
	// w := FileWatcher("/root/project", nil)
	w := FileWatcher(UpperDir, nil)

	// 开启 Gossip 同步
	nodeInfo, err := StartGossip(functionName)
	if err != nil {
		fmt.Printf("无法开启 Gossip 同步: %v\n", err)
	}

	// 程序关闭时取消文件系统监控并删除节点信息
	quit := make(chan os.Signal, 1)
	defer close(quit)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case q := <-quit: // 正常退出
		fmt.Printf("获取到信号%s\n", q)
		w.watch.Close()
		DeleteNode(nodeInfo)
	case <-time.After(10 * time.Minute): // 超时退出
		fmt.Println("运行超时，程序退出")
		w.watch.Close()
		DeleteNode(nodeInfo)
		os.Exit(0)
	}
}
