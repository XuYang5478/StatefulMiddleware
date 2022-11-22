package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
)

func getUpperDir(containerId string) string {
	// 连接至容器服务
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		fmt.Println("无法连接至contained。", err)
		os.Exit(-1)
	}
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), "k8s.io")

	container, err1 := client.ContainerService().Get(ctx, containerId)
	if err1 != nil {
		fmt.Println("无法获取容器信息。", err1.Error())
		os.Exit(-1)
	}

	mounts, err2 := client.SnapshotService(containerd.DefaultSnapshotter).Mounts(ctx, container.SnapshotKey)
	if err2 != nil {
		fmt.Println("无法获取容器挂载信息。", err2.Error())
		os.Exit(-1)
	}

	// 获取容器文件系统的挂载信息
	var upperdir string
	for _, mount := range mounts {
		for _, s := range mount.Options {
			fmt.Println(s)
			if strings.Contains(s, "upperdir") {
				upperdir = strings.Split(s, "=")[1]
			}
		}
	}
	fmt.Println("upperdir:", upperdir)
	return upperdir
}
