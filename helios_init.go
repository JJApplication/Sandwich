package main

import (
	"context"
	"fmt"
	"sandwich/constant"
	"sandwich/log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitHeliosConfig() {
	// 连接到Unix域套接字
	log.InfoF("Start to connect to %s\n", constant.HeliosAddress)
	conn, err := grpc.NewClient(
		fmt.Sprintf("unix://%s", constant.HeliosAddress),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.ErrorF("Failed to connect unix: %v\n", err)
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
		log.ErrorF("Failed to get server info: %v\n", err)
		return
	}

	// 打印结果
	log.Debug("Server Info:\n")
	log.DebugF("Host: %s\n", resp.Host)
	log.DebugF("Port: %d\n", resp.Port)
	log.DebugF("Internal Flag: %s\n", resp.InternalFlag)
	log.DebugF("Internal Local Flag: %s\n", resp.InternalLocalFlag)
	log.DebugF("Internal Backend Flag: %s\n", resp.InternalBackendFlag)

	// 刷新值
	constant.FrontendHost = resp.Host
	constant.FrontendPort = int(resp.Port)
	constant.FrontendFlag = resp.InternalFlag
	constant.BackendHeader = resp.InternalLocalFlag
	constant.ProxyApp = resp.InternalBackendFlag
	log.Info("Init Helios Config")
}
