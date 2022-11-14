# StatefulMiddleware
用 Go 编写的一个运行在 openwhisk 集群每个节点上的程序，为 serverless 服务提供状态管理支持。

## TODO
- [ ] containerd 相关的 cmd 指令，包括：
  - check 镜像是否存在
  - commit 镜像
- [ ] 监听 checkpoint 目录，同步 commit 镜像
- [ ] 容器启动后，监听容器的 upperdir，同步运行状态