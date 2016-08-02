package main

import (
	"container/list"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
)

type Router struct {
	r_conn      net.Conn
	client_conn *list.List
}

var R map[string]Router

func CheckErr_exit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func CheckErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error:", err)
	}
}

func routerRecv(c net.Conn) {
	var buf [1024]byte
	var l *list.List
	var router Router
	res, e := c.Read(buf[0:])
	if e != nil {
		CheckErr(e)
		return
	}
	defer func() {
		c.Close()
	}()
	if res > 0 {
		fmt.Printf("%s\n", string(buf[0:12]))
		router.r_conn = c
		l = list.New()
		router.client_conn = l
		R[string(buf[0:12])] = router
		defer func() {
			delete(R, string(buf[0:12]))
		}()
	} else {
		return
	}
	s := "889e0d7343405c079195e7b8903c8c9e\n"
	t := "b0061974914468de549a2af8ced10316\n"
	var start int
	var end int
	for {
		res, e := c.Read(buf[0:])
		//fmt.Println(buf[:])
		if e != nil || res == 0 {
			c.Close()
			return
		}
		start = 0
		end = res
		n1 := strings.Index(string(buf[start:end]), s) //1
		n2 := strings.Index(string(buf[start:end]), t) //1
		if n1 >= 0 {
			start = n1 + len(s)
		}
		if n2 >= 0 {
			end = n2
		}

		//		fmt.Println("router ", string(buf[start:end]))
		//		fmt.Println("router1", n1, n2, start, end, "\n\n\n\n\n")
		if end-start > 0 {
			ll := l.Back()
			if ll != nil {
				cli := ll.Value.(net.Conn)
				_, err := cli.Write(buf[start:end])
				if err != nil {
					fmt.Fprintf(os.Stderr, "error:", err)
					l.Remove(ll)
				}
			}
		}
		if n2 >= 0 {
			ll := l.Back()
			if ll != nil {
				cli := ll.Value.(net.Conn)
				cli.Close()
				l.Remove(ll)
			}
		}

	}
}

func routerWrite(c net.Conn) {
	defer func() {
		//fmt.Println("client exit")
		c.Close()
	}()
	tcp, ok := c.(*net.TCPConn)
	if !ok {
		return
	}
	err := tcp.SetKeepAlive(true)
	if err != nil {
		return
	}

	var buf [1024]byte
	res, e := c.Read(buf[0:])
	if e != nil {
		CheckErr(e)
		return
	}
	var mac string
	var cmd string
	if res > 0 {
		n1 := strings.Index(string(buf[0:]), "mac=") //1
		n2 := strings.Index(string(buf[0:]), "cmd=") //1
		//		fmt.Println(string(buf[:]))
		//		fmt.Println(n1, n2)
		if n1 >= 0 && n2 > 0 {
			mac = string(buf[(n1 + 4) : n1+4+12])
			cmd = string(buf[n2+4:])
			if len(cmd) <= 0 {
				return
			}
		} else {
			return
		}
	}
	fmt.Printf("client %s\n", mac)
	fmt.Printf("client %s\n", cmd)
	if _, ok := R[mac]; ok == false {
		c.Write([]byte("this devices is not online\n"))
		return
	}
	r := R[mac].r_conn
	fmt.Print(cmd)
	if strings.Index(cmd, "RouterStop") >= 0 {
		r.Close()
		return
	}
	s := "echo 889e0d7343405c079195e7b8903c8c9e\n"
	n, err := r.Write([]byte(s))
	if n != len(s) || err != nil {
		return
	}
	//	fmt.Println(cmd)
	n, e = r.Write([]byte(cmd))
	if n != len(cmd) || err != nil {
		return
	}
	t := "echo b0061974914468de549a2af8ced10316\n"
	n, err = r.Write([]byte(t))
	if n != len(t) || err != nil {
		return
	}

	l := R[mac].client_conn
	l.PushBack(c)
	for {
		_, e := c.Read(buf[0:])
		if e != nil {
			return
		}
	}
}

func main() {
	R = make(map[string]Router)
	go func() {
		R = make(map[string]Router)
		cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
		CheckErr_exit(err)
		port := ":1299"
		//		config := tls.Config{Certificates: []tls.Certificate{cert}}
		//		config.Rand = rand.Reader
		//		server, err := tls.Listen("tcp", port, &config)
		config := tls.Config{Certificates: []tls.Certificate{cert}}
		config.Rand = rand.Reader
		server, err := tls.Listen("tcp", port, &config)
		CheckErr_exit(err)

		for {
			conn, err := server.Accept()
			if err != nil {
				continue
			}
			go routerRecv(conn)
		}
	}()

	port := ":1200"
	tcpAddr, err := net.ResolveTCPAddr("tcp4", port)
	CheckErr_exit(err)
	server, err := net.ListenTCP("tcp", tcpAddr)
	CheckErr_exit(err)
	for {
		conn, err := server.Accept()
		if err != nil {
			continue
		}
		go routerWrite(conn)
	}
}
