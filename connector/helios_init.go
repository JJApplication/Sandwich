package connector

import (
	"context"
	"fmt"
	"sandwich/config"
	"sandwich/log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitHeliosConfig() {
	cf := config.Get()
	// 连接到Unix域套接字
	log.GetLogger().Info().Str("address", cf.FrontProxy.GrpcAddr).Msg("Start to connect")
	conn, err := grpc.NewClient(
		fmt.Sprintf("unix://%s", cf.FrontProxy.GrpcAddr),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.GetLogger().Error().Err(err).Msg("Failed to connect unix")
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
		log.GetLogger().Error().Err(err).Msg("Failed to get server info")
		return
	}

	// 打印结果
	log.GetLogger().Debug().
		Str("Host", resp.Host).
		Int32("Port", resp.Port).
		Str("Internal Flag", resp.InternalFlag).
		Str("Internal Local Flag", resp.InternalLocalFlag).
		Str("Internal Backend Flag", resp.InternalBackendFlag).
		Msg("Server Info")

	// 刷新值
	cf.FrontProxy.FrontendHost = resp.Host
	cf.FrontProxy.FrontendPort = int(resp.Port)
	cf.FrontProxy.FrontendFlag = resp.InternalFlag
	cf.ProxyHeader.BackendHeader = resp.InternalLocalFlag
	cf.ProxyHeader.ProxyApp = resp.InternalBackendFlag
	log.Info("Init Helios Config")
}
