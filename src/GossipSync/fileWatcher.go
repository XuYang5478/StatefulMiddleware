package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

func noneSysFileOrDir(path string) bool {
	if strings.Contains(path, "/proc") || strings.Contains(path, "/dev") ||
		strings.Contains(path, "/sys") || strings.Contains(path, "/run") ||
		strings.Contains(path, "/etc") {
		return false
	}
	return true
}

type Watch struct {
	watch *fsnotify.Watcher
}

// 递归遍历为子目录添加监控
func (w *Watch) watchDir(dir string, db *sql.DB) {
	filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		//这里判断是否为目录，只需监控目录即可
		//目录下的文件也在监控范围内，不需要我们一个一个加
		if info.IsDir() && noneSysFileOrDir(path) {
			dir_path, err1 := filepath.Abs(path)
			if err1 != nil {
				return err1
			}
			err2 := w.watch.Add(dir_path)
			if err2 != nil {
				return err2
			}
			fmt.Println("添加监控：", dir_path)
		}
		return nil
	})

	go func() {
		for {
			select {
			case ev := <-w.watch.Events:
				{
					if ev.Op&fsnotify.Create == fsnotify.Create {
						// fmt.Println("创建文件 : ", ev.Name)
						//这里获取新创建文件的信息，如果是目录，则加入监控中
						fi, err := os.Stat(ev.Name)
						if err == nil {
							if fi.IsDir() {
								w.watch.Add(ev.Name)
								fmt.Println("添加监控 : ", ev.Name)
							}
							addGossipMsg("c", ev.Name, fmt.Sprint(fi.ModTime().Unix()))

						}
					}
					if ev.Op&fsnotify.Write == fsnotify.Write {
						// fmt.Println("写入文件 : ", ev.Name)
						fi, err := os.Stat(ev.Name)
						if err == nil {
							addGossipMsg("c", ev.Name, fmt.Sprint(fi.ModTime().Unix()))
						}
					}
					if ev.Op&fsnotify.Remove == fsnotify.Remove {
						// fmt.Println("删除文件 : ", ev.Name)
						//如果删除文件是目录，则移除监控
						// go 无法获取被删除的文件/目录的信息
						// 所以直接删除对其的监控
						w.watch.Remove(ev.Name)
						addGossipMsg("d", ev.Name, fmt.Sprint(time.Now().Unix()))
					}
					if ev.Op&fsnotify.Rename == fsnotify.Rename {
						// fmt.Println("重命名文件 : ", ev.Name)
						//如果重命名文件是目录，则移除监控
						//注意这里无法使用os.Stat来判断是否是目录了
						//因为重命名后，go已经无法找到原文件来获取信息了
						//所以这里就简单粗爆的直接remove好了
						// fmt.Println("删除监控 : ", ev.Name)
						w.watch.Remove(ev.Name)
					}
					// if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
					// 	fmt.Println("修改权限 : ", ev.Name)
					// }
				}
			case err := <-w.watch.Errors:
				{
					fmt.Println("error : ", err)
					return
				}
			}
		}
	}()
}

func FileWatcher(path string, db *sql.DB) Watch {
	watch, _ := fsnotify.NewWatcher()
	w := Watch{
		watch: watch,
	}
	w.watchDir(path, db)

	return w
}
