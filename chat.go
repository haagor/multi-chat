package main

import (
    "net"
    "bufio"
    "log"
    "fmt"
    "strings"
    "time"
)


var dico = make(map[string]net.Conn)    //association de [nickname]net.Conn

func handleConnection(conn net.Conn, mapConn chan map[string]net.Conn, global chan string) {
    defer conn.Close()
    defer fmt.Fprintf(conn, "Idle for a too long time. I disconnect you!\n");

    reader := bufio.NewReader(conn)
    fmt.Fprintf(conn, "Nickname? ");
    nickname,err := reader.ReadString('\n')
    if err != nil {
        log.Println(err)
        return
    }
    fmt.Fprintf(conn, "Welcome on board %s", nickname);

    connect := "connect;" + nickname + ";"
    global <- connect
    c := make(map[string]net.Conn)
    c[nickname] = conn
    mapConn <- c

    defer func() {
        disconnect := "disconnect;" + nickname
        global <- disconnect
    } ()

    timer := time.NewTimer(time.Second * 15)
    timeout := make(chan int)

    go func() {
        <-timer.C
        message := "time;" + nickname
        global <- message
        timeout <- 1
    } ()

    go func() {
        for {
            line,err := reader.ReadString('\n')
            if err != nil {
                //log.Println(err)
                return
            }
            line = "msg;" + nickname + ";" + line
            global <- line
            timer.Reset(time.Second * 15)
        }
    } ()

    <- timeout
}

// goroutine unique qui va écouter sur le chan global et rediriger les bons messages en connection direct 
func listen_chat(mapConn chan map[string]net.Conn, global chan string) {
    for {
        go addDico(mapConn)

        line := <- global
        split := strings.Split(line, ";")

        switch split[0] {
        case "msg" :
            message := split[1][:len(split[1])-1] + ": " + split[2]
            broadcast(split[1], message)
        case "connect" :
            message := split[1][:len(split[1])-1] + " has joined.\n"
            broadcast(split[1], message)
            fmt.Printf("Login of %s\n", split[1][:len(split[1])-1])
        case "disconnect" :
            message := split[1][:len(split[1])-1] + " is gone.\n"
            broadcast(split[1], message)
            delete(dico, split[1][:len(split[1])-1])
            fmt.Printf("Logout of %s\n", split[1][:len(split[1])-1])
        case "time" :
            message := split[1][:len(split[1])-1] + " was idle to long and was disconnected.\n"
            broadcast(split[1], message)
            fmt.Printf("%s seems to be out. Force disconnection.\n", split[1][:len(split[1])-1])
        }
    }
}

func broadcast(sendBy string, message string) {
    for key, value := range dico {
        if key != sendBy {
            fmt.Fprintf(value, message);
        }
    }
}

func addDico(mapConn chan map[string]net.Conn) {
    for {
        m := <- mapConn
        for k, v := range m {
            dico[k] = v
        }
    }
}

func main() {
    listener, err := net.Listen("tcp", "localhost:1234")
    if err != nil {
        log.Fatal(err)
    }

    global := make(chan string)
    mapConn := make(chan map[string]net.Conn) // ce chan ser à maintenir l'association [nickanme]net.Conn
    go listen_chat(mapConn, global)

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Println(err)
            continue
        }
        go handleConnection(conn, mapConn, global)
    }
}