package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type containerDataJson struct {
	Id, ImageName, ActionName string
}

var RunningGossip = map[string]int{}

func RunNerdctl(cmd string) string {
	nerdCmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("nerdctl -n k8s.io %s", cmd))
	out, err := nerdCmd.CombinedOutput()
	if err != nil {
		log.Println("[error] cmd 执行错误：", err.Error())
		return ""
	}
	if string(out) == "" {
		return "ok"
	} else {
		return string(out)
	}

}

func CheckImageHandler(w http.ResponseWriter, r *http.Request) {
	var data containerDataJson
	err1 := json.NewDecoder(r.Body).Decode(&data)
	if err1 != nil {
		log.Println("[error] 解析参数出错:", err1.Error())
		w.Write([]byte("error"))
		return
	}

	log.Println("[info] 接收到检查镜像请求。image: ", data.ImageName)

	result := true
	nodes := GetNodes(SySFuncName)
	if len(nodes) > 0 {
		for _, node := range nodes {
			cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("ssh root@%s nerdctl -n k8s.io image ls %s", node.Host, data.ImageName))
			output, err2 := cmd.CombinedOutput()
			if err2 != nil {
				result = false
				log.Println("[error] 命令执行出错：", err2.Error())
				break
			}
			fmt.Println(string(output))
			fmt.Println(len(strings.Split(string(output), "\n")))
			if len(strings.Split(string(output), "\n")) <= 2 {
				result = false
				log.Printf("[warning] %s 上没有找到镜像 %s\n", node.Host, data.ImageName)
				break
			}
		}
	} else {
		result = false
	}

	log.Printf("[info] %s 查找结果：%v\n", data.ImageName, result)

	if result {
		w.Write([]byte("yes"))
	} else {
		w.Write([]byte("no"))
	}
}

func CommitImageHandler(w http.ResponseWriter, r *http.Request) {
	var data containerDataJson
	err1 := json.NewDecoder(r.Body).Decode(&data)
	if err1 != nil {
		log.Println("[error] 解析参数出错:", err1.Error())
		w.Write([]byte("error"))
		return
	}

	log.Println("[info] 接收到 commit 镜像请求。containerId:", data.Id, "image:", data.ImageName)
	commit := RunNerdctl(fmt.Sprintf("commit -p=false %s %s", data.Id, data.ImageName))
	log.Println("[info] commit 执行结果：", commit)

	names := strings.FieldsFunc(data.ImageName, func(r rune) bool {
		if r == '/' || r == ':' {
			return true
		}
		return false
	})

	createTime := time.Now().Unix()
	imageFileName := fmt.Sprintf("/root/checkpoint/%s_%s.tar", names[1], fmt.Sprint(createTime))

	log.Println("[info] 导出镜像：", data.ImageName, "->", imageFileName)
	save := RunNerdctl(fmt.Sprintf("save %s -o %s", data.ImageName, imageFileName))
	log.Println("[info] save 执行结果：", save)

	AddGossipMsg("c", imageFileName, fmt.Sprint(createTime))

	if commit != "" && save != "" {
		w.Write([]byte("ok"))
	} else {
		w.Write([]byte("error"))
	}
}

func StartContainerHandler(w http.ResponseWriter, r *http.Request) {
	var data containerDataJson
	err1 := json.NewDecoder(r.Body).Decode(&data)
	if err1 != nil {
		log.Println("[error] 解析参数出错:", err1.Error())
		w.Write([]byte("error"))
		return
	}

	names := strings.FieldsFunc(data.ActionName, func(r rune) bool {
		if r == '_' || r == '-' {
			return true
		}
		return false
	})

	log.Printf("[info] 容器 (%s, %s) 启动，准备开启 gossip 服务\n", names[0], data.Id)
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("/usr/bin/GossipSync %s %s", names[0], data.Id))
	cmd.Stdout = log.Writer()
	err := cmd.Start()
	if err != nil {
		log.Println("[error] 无法启动 gossip 服务:", err.Error())
		w.Write([]byte("error"))
		return
	}

	RunningGossip[data.Id] = cmd.Process.Pid
	w.Write([]byte("ok"))
}

func StopContainerHandler(w http.ResponseWriter, r *http.Request) {
	var data containerDataJson
	err1 := json.NewDecoder(r.Body).Decode(&data)
	if err1 != nil {
		log.Println("[error] 解析参数出错:", err1.Error())
		w.Write([]byte("error"))
		return
	}

	log.Printf("[info] 容器 (%s, %s) 停止，准备关闭 gossip 服务\n", data.ActionName, data.Id)

	// kill 时会将父进程一起 kill 掉
	killCmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("kill %d", RunningGossip[data.Id]))
	out, err := killCmd.CombinedOutput()
	if err != nil {
		log.Println("[error] cmd 执行错误：", err.Error())
		w.Write([]byte("error"))
		return
	}
	log.Println("[info] kill 执行结果：", out)

	//result := RunNerdctl(fmt.Sprintf("commit -p=false %s %s", data.Id, data.ImageName))
	//log.Println("[info] commit 执行结果：", result)
	w.Write([]byte("ok"))
}
