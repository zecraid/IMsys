package main

import (
	"net"
	"strings"
)

type User struct {
	Name    string
	Addr    string
	Channel chan string
	conn    net.Conn
	server  *Server
}

// 创建一个用户的方法
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name:    userAddr,
		Addr:    userAddr,
		Channel: make(chan string),
		conn:    conn,
		server:  server,
	}

	//启动监听当前user channel的goruntine
	go user.ListenMessage()
	return user
}

// 用户上线业务
func (user *User) Online() {
	//用户上线，将用户加入到OnlineMap中
	user.server.mapLock.Lock()
	user.server.OnlineMap[user.Name] = user
	user.server.mapLock.Unlock()
	//广播当前用户上线消息
	user.server.BroadCast(user, "已上线")
}

// 用户下线业务
func (user *User) Offline() {
	//用户上线，将用户从OnlineMap中删除
	user.server.mapLock.Lock()
	delete(user.server.OnlineMap, user.Name)
	user.server.mapLock.Unlock()
	//广播当前用户上线消息
	user.server.BroadCast(user, "已下线")
}

// 给当前用户对应的客户端发送消息
func (user *User) sendMsg(msg string) {
	user.conn.Write([]byte(msg))
}

// 用户处理消息业务
func (user *User) DoMessage(msg string) {
	if msg == "who" {
		//查询当前在线用户都有哪些
		user.server.mapLock.Lock()
		for _, tempuser := range user.server.OnlineMap {
			onlineMap := "[" + tempuser.Addr + "]" + tempuser.Name + ":在线...\n"
			user.sendMsg(onlineMap)
		}
		user.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		//消息格式：rename|张三
		newName := strings.Split(msg, "|")[1]
		//判断name是否存在
		_, ok := user.server.OnlineMap[newName]
		if ok { //查询成功
			user.sendMsg("当前用户名被使用")
		} else {
			user.server.mapLock.Lock()
			delete(user.server.OnlineMap, user.Name)
			user.server.OnlineMap[newName] = user
			user.server.mapLock.Unlock()
			user.Name = newName
			user.sendMsg("您已经更新用户名:" + user.Name + "\n")
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		//消息格式：to｜张三｜消息内容
		//1、获取对方的用户名
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			user.sendMsg("消息格式不准确，请使用'to｜张三｜消息内容'格式\n")
		}
		//2、根据用户名，得到对方的User对象
		remoteUser, ok := user.server.OnlineMap[remoteName]
		if !ok {
			user.sendMsg("该用户名不存在\n")
			return
		}
		//3、获取消息内容，通过对方的User对象将消息内容发送过去
		content := strings.Split(msg, "|")[2]
		if content == "" {
			user.sendMsg("无消息内容，请重新发送\n")
			return
		}
		remoteUser.sendMsg(user.Name + "对您说：" + content + "\n")
	} else {
		user.server.BroadCast(user, msg)
	}
}

func (user *User) ListenMessage() {
	for {
		msg := <-user.Channel
		user.conn.Write([]byte(msg + "\n"))
	}
}
