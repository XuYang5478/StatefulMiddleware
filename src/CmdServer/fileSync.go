package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func RunCmd(cmd *exec.Cmd) {
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("[error] Error: %s 输出: %s", err.Error(), output)
	} else {
		log.Println("[info] 指令输出：", output)
	}
}

func ModifiedFile(filePath string, localHost string, remoteHost string) {
	if localHost != remoteHost {
		var cmd *exec.Cmd
		remotePath := fmt.Sprintf("root@%s:%s", remoteHost, filePath)
		cmd = exec.Command("scp", "-p", remotePath, filePath)
		log.Printf("[info] 更新：%s -> %s\n", remotePath, filePath)

		RunCmd(cmd)

		if strings.HasSuffix(filePath, ".tar") {
			load := RunNerdctl("load -i " + filePath)
			log.Printf("[info] 镜像导入执行结果：%s\n", load)
			// 导入后把镜像文件删除，节省空间
			del := exec.Command("rm", filePath)
			RunCmd(del)
		}
	}
}

func DeleteFile(filePath string, localHost string, remoteHost string) {
	if localHost != remoteHost {
		cmd := exec.Command("rm", "-rf", filePath)
		log.Printf("[info] 删除：%s\n", filePath)
		RunCmd(cmd)
	}
}
