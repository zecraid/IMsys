package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	//在线用户的列表
	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	//消息广播的channel
	Message chan string
}

// 创建一个server的接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

// 监听Message广播消息的channel的goruntine，一旦有消息就发送给全部的在线User
func (server *Server) ListenMessage() {
	for {
		msg := <-server.Message
		//将msg发送给全部的在线User
		server.mapLock.Lock()
		for _, client := range server.OnlineMap {
			client.Channel <- msg
		}
		server.mapLock.Unlock()
	}
}

// 广播消息的方法
func (server *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	server.Message <- sendMsg
}

func (server *Server) Handler(conn net.Conn) {
	//...当前连接的业务
	//fmt.Println("连接建立成功")
	user := NewUser(conn, server)
	user.Online()

	//监听是否活跃的channel
	isLive := make(chan bool)

	//接受客户端发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err:", err)
				return
			}
			//提取用户的消息（去除'\n'）
			msg := string(buf[:n-1])

			//用户针对msg进行消息处理
			user.DoMessage(msg)

			//用户的任意消息，代表当前的用户是一个活跃的
			isLive <- true
		}
	}()

	//当前handler阻塞
	for {
		select {
		case <-isLive:
			//当前用户是活跃的，应该重置定时器
			//不做任何事情，为了激活select，更新下面的定时器
		case <-time.After(time.Hour * 1):
			//已经超时
			//将当前的User强制关闭
			user.sendMsg("你被踢了")
			//销毁用的资源
			close(user.Channel)
			//关闭连接
			conn.Close()
			//退出当前的Handler
			return //runtime.Goexit()
		}
	}
}

// 启动服务器的方法
func (server *Server) Strat() {
	//socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", server.Ip, server.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	//close listen socket
	defer listener.Close()

	//启动监听Message的goruntine
	go server.ListenMessage()
	for {
		//accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}
		//do handler
		go server.Handler(conn)
	}

}
