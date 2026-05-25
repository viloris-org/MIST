//go:build !linux

package main

import (
	"context"
	"mist/mistclient"
	"net"

	"github.com/sirupsen/logrus"
)

func handleRedirectConnection(ctx context.Context, conn net.Conn, client *mistclient.Client) {
	conn.Close()
	logrus.Errorln("redirect inbound is only supported on linux")
}
