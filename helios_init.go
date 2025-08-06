package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitHeliosConfig() {
	// 连接到Unix域套接字
	log.Printf("Start to connect to %s\n", HeliosAddress)
	conn, err := grpc.NewClient(
		fmt.Sprintf("unix://%s", HeliosAddress),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("Failed to connect unix: %v\n", err)
		return
	}
	defer conn.Close()

	// 创建客户端
	client := NewHeliosServiceClient(conn)

	// 调用GetServerInfo方法
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetServerInfo(ctx, &GetServerInfoRequest{})
	if err != nil {
		log.Printf("Failed to get server info: %v\n", err)
		return
	}

	// 打印结果
	debug("Server Info:\n")
	debugF("Host: %s\n", resp.Host)
	debugF("Port: %d\n", resp.Port)
	debugF("Internal Flag: %s\n", resp.InternalFlag)
	debugF("Internal Local Flag: %s\n", resp.InternalLocalFlag)
	debugF("Internal Backend Flag: %s\n", resp.InternalBackendFlag)

	// 刷新值
	FrontendHost = resp.Host
	FrontendPort = int(resp.Port)
	FrontendFlag = resp.InternalFlag
	BackendHeader = resp.InternalLocalFlag
	ProxyApp = resp.InternalBackendFlag
	log.Println("Init Helios Config")
}
