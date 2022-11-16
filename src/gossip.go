package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/hashicorp/memberlist"
)

// 文件状态信息
type fileInfo struct {
	Action    string // create==modify delete
	Host      string // 最后更新文件的节点
	FilePath  string // 镜像文件路径
	Timestamp string //最后更新的时间戳
}

var (
	mtx        sync.RWMutex
	files      = map[string]fileInfo{}
	broadcasts *memberlist.TransmitLimitedQueue
	LocalHost  = "" // 本机 IP
)

type broadcast struct {
	msg    []byte
	notify chan<- struct{}
}

type delegate struct{}

// 文件更新信息
// update == fileInfo

func (b *broadcast) Invalidates(other memberlist.Broadcast) bool {
	return false
}

func (b *broadcast) Message() []byte {
	return b.msg
}

func (b *broadcast) Finished() {
	if b.notify != nil {
		close(b.notify)
	}
}

func (d *delegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d *delegate) NotifyMsg(b []byte) {
	if len(b) == 0 {
		return
	}

	var u fileInfo
	if err := json.Unmarshal(b, &u); err != nil {
		log.Printf("[error] 解析 gossip 消息出错: %v\n", err)
		return
	}
	log.Println("[info] gossip 接收到: ", u)
	mtx.Lock()
	switch u.Action {
	case "c":
		localInfo, ok := files[u.FilePath]
		if !ok || localInfo.Timestamp < u.Timestamp {
			files[u.FilePath] = u
			log.Println("NotifyMsg 更新文件：", u.FilePath)
			// 将镜像文件同步过来
			go ModifiedFile(u.FilePath, LocalHost, u.Host)
		}
	case "d":
		_, ok := files[u.FilePath]
		if ok {
			delete(files, u.FilePath)
			log.Println("NotifyMsg 删除文件：", u.FilePath)
			// 同步删除本地镜像文件
			go DeleteFile(u.FilePath, LocalHost, u.Host)
		}
	default:
		fmt.Println("NotifyMsg 未知动作：", u.Action)
	}
	mtx.Unlock()
}

func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return broadcasts.GetBroadcasts(overhead, limit)
}

func (d *delegate) LocalState(join bool) []byte {
	mtx.RLock()
	m := files
	mtx.RUnlock()
	b, _ := json.Marshal(m)
	return b
}

func (d *delegate) MergeRemoteState(buf []byte, join bool) {
	if len(buf) == 0 {
		return
	}
	if !join {
		return
	}
	var m map[string]fileInfo
	if err := json.Unmarshal(buf, &m); err != nil {
		log.Println("[error] 无法合并状态")
		return
	}
	mtx.Lock()
	for file, info := range m {
		localInfo, ok := files[file]
		if !ok || localInfo.Timestamp < info.Timestamp {
			files[file] = info
			log.Println("[info] MergeRemoteState 更新文件：", file)
			go ModifiedFile(info.FilePath, LocalHost, info.Host)
		}
	}
	mtx.Unlock()
}

type eventDelegate struct{}

func (ed *eventDelegate) NotifyJoin(node *memberlist.Node) {
	fmt.Println("A node has joined: " + node.String())
}

func (ed *eventDelegate) NotifyLeave(node *memberlist.Node) {
	fmt.Println("A node has left: " + node.String())
}

func (ed *eventDelegate) NotifyUpdate(node *memberlist.Node) {
	fmt.Println("A node was updated: " + node.String())
}

func AddGossipMsg(action string, target string, timestmap string) {

	// 将文件改动信息保存到本地状态
	files[target] = fileInfo{
		Action:    action,
		Host:      LocalHost,
		FilePath:  target,
		Timestamp: timestmap,
	}

	b, err := json.Marshal(files[target])

	if err != nil {
		log.Printf("[error] 创建 gossip 消息出错：%v\n", err)
	}

	// 文件改动信息广播给其他节点
	broadcasts.QueueBroadcast(&broadcast{
		msg:    b,
		notify: nil,
	})
}

func StartGossip(funcName string) (CouchReturnDocument, error) {
	hostname, _ := os.Hostname()
	c := memberlist.DefaultLocalConfig()
	c.Events = &eventDelegate{}
	c.Delegate = &delegate{}
	c.BindPort = 0
	id, _ := uuid.NewUUID()
	c.Name = hostname + "-" + id.String()
	m, err := memberlist.Create(c)
	if err != nil {
		return CouchReturnDocument{}, err
	}
	nodes := GetNodes(funcName)
	if len(nodes) > 0 {
		all := []string{}
		for _, n := range nodes {
			all = append(all, fmt.Sprintf("%s:%s", n.Host, n.Port))
		}
		_, err := m.Join(all)
		if err != nil {
			log.Printf("[error] 无法加入 Gossip 集群: %v\n", err)
			return CouchReturnDocument{}, err
		}
	}

	broadcasts = &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return m.NumMembers()
		},
		RetransmitMult: 3,
	}
	node := m.LocalNode()
	log.Printf("[info] Local node %s:%d\n", node.Addr, node.Port)
	LocalHost = fmt.Sprint(node.Addr)

	// 将运行该函数的节点数据保存到数据库中
	saveResult := SaveNode(funcName, fmt.Sprint(node.Addr), fmt.Sprint(node.Port))

	return saveResult, nil
}
