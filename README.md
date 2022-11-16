# StatefulMiddleware
用 Go 编写的一个运行在 openwhisk 集群每个节点上的程序，为 serverless 服务提供状态管理支持。

## TODO
- [x] containerd 相关的 cmd 指令，包括：
  - check 镜像是否存在
  - commit 镜像
  - start 容器启动，为该容器开启 gossip 服务
  - stop 容器停止，关闭 gossip 服务
- [x] 监听 checkpoint 目录，同步 commit 镜像