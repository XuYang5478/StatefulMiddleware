package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/hashicorp/memberlist"
)

// 文件状态信息
type fileInfo struct {
	Host      string // 最后更新文件的节点
	Upperdir  string // 容器文件系统根目录
	Timestamp string //最后更新的时间戳
}

var (
	mtx        sync.RWMutex
	files      = map[string]fileInfo{} // 文件更改列表
	broadcasts *memberlist.TransmitLimitedQueue
	LocalHost  = "" // 本机 IP
)

type broadcast struct {
	msg    []byte
	notify chan<- struct{}
}

type delegate struct{}

// 文件更新信息
type update struct {
	Action   string // c_reate(modify), d_elete，文件动作
	Host     string
	Upperdir string
	Data     map[string]string // file/dir - timestamp 产生动作的文件路径和时间戳
}

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

	var u update
	if err := json.Unmarshal(b, &u); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println("Receive: ", u)
	mtx.Lock()
	for k, v := range u.Data {
		actuallyFile := GetRelativePath(k)
		switch u.Action {
		case "c":
			localInfo, ok := files[actuallyFile]
			if !ok || localInfo.Timestamp < v {
				files[actuallyFile] = fileInfo{
					Host:      u.Host,
					Upperdir:  u.Upperdir,
					Timestamp: v,
				}
				fmt.Println("NotifyMsg 更新文件：", actuallyFile)
				go ModifiedFile(LocalHost, UpperDir, u.Host, k)
			}
		case "d":
			_, ok := files[actuallyFile]
			if ok {
				delete(files, actuallyFile)
				fmt.Println("NotifyMsg 删除文件：", actuallyFile)
				go DeleteFile(UpperDir, k)
			}
		default:
			fmt.Println("NotifyMsg 未知动作：", u.Action)
		}
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
		fmt.Println("Error: 无法合并状态")
		return
	}
	mtx.Lock()
	for file, info := range m {
		localInfo, ok := files[file]
		if !ok || localInfo.Timestamp < info.Timestamp {
			files[file] = info
			fmt.Println("MergeRemoteState 更新文件：", file)
			go ModifiedFile(LocalHost, UpperDir, info.Host, info.Upperdir+file)
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

func addGossipMsg(action string, target string, timestmap string) {
	actuallyFile := GetRelativePath(target)

	// 将文件改动信息保存到本地状态
	files[actuallyFile] = fileInfo{
		Host:      LocalHost,
		Upperdir:  UpperDir,
		Timestamp: timestmap,
	}

	b, err := json.Marshal(update{
		Action:   action,
		Host:     LocalHost,
		Upperdir: UpperDir,
		Data: map[string]string{
			target: timestmap,
		},
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
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
	// if len(nodes) > 0 {
	// 	parts := strings.Split(nodes, ",")
	// 	_, err := m.Join(parts)
	// 	if err != nil {
	// 		fmt.Printf("Error: %v\n", err)
	// 		return err
	// 	}
	// }
	nodes := GetNodes(funcName)
	if len(nodes) > 0 {
		all := []string{}
		for _, n := range nodes {
			all = append(all, fmt.Sprintf("%s:%s", n.Host, n.Port))
		}
		_, err := m.Join(all)
		if err != nil {
			fmt.Printf("无法加入 Gossip 集群: %v\n", err)
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
	fmt.Printf("Local node %s:%d\n", node.Addr, node.Port)
	LocalHost = fmt.Sprint(node.Addr)

	// 将运行该函数的节点数据保存到数据库中
	saveResult := SaveNode(funcName, fmt.Sprint(node.Addr), fmt.Sprint(node.Port))

	return saveResult, nil
}
