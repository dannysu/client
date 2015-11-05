// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package client

import (
	"time"

	"golang.org/x/net/context"

	"github.com/keybase/cli"
	"github.com/keybase/client/go/libcmdline"
	"github.com/keybase/client/go/libkb"
	keybase1 "github.com/keybase/client/go/protocol"
	rpc "github.com/keybase/go-framed-msgpack-rpc"
)

type CmdShowNotifications struct {
	libkb.Contextified
}

func (c *CmdShowNotifications) ParseArgv(ctx *cli.Context) error {
	return nil
}

func (c *CmdShowNotifications) Run() error {
	_, err := GlobUI.Println("Showing notifications:")
	if err != nil {
		return err
	}

	display := newNotificationDisplay(c.G())

	protocols := []rpc.Protocol{
		keybase1.NotifySessionProtocol(display),
		keybase1.NotifyUsersProtocol(display),
	}
	if err := RegisterProtocols(protocols); err != nil {
		return err
	}
	cli, err := GetNotifyCtlClient(c.G())
	if err != nil {
		return err
	}
	if err := cli.ToggleNotifications(context.TODO(), keybase1.NotificationChannels{Session: true, Users: true}); err != nil {
		return err
	}

	_, err = GlobUI.Println("waiting for notifications...")
	if err != nil {
		return err
	}
	for {
		time.Sleep(time.Second)
	}
}

func NewCmdShowNotifications(cl *libcmdline.CommandLine, g *libkb.GlobalContext) cli.Command {
	return cli.Command{
		Name:  "show-notifications",
		Usage: "Display all notifications sent by daemon in real-time",
		Action: func(c *cli.Context) {
			cl.ChooseCommand(&CmdShowNotifications{Contextified: libkb.NewContextified(g)}, "show-notifications", c)
		},
		Flags: []cli.Flag{},
	}
}

func (c *CmdShowNotifications) GetUsage() libkb.Usage {
	return libkb.Usage{}
}

type notificationDisplay struct {
	libkb.Contextified
}

func newNotificationDisplay(g *libkb.GlobalContext) *notificationDisplay {
	return &notificationDisplay{Contextified: libkb.NewContextified(g)}
}

func (d *notificationDisplay) printf(fmt string, args ...interface{}) error {
	_, err := d.G().UI.GetTerminalUI().Printf(fmt, args...)
	return err
}

func (d *notificationDisplay) LoggedOut(_ context.Context) error {
	return d.printf("Logged out\n")
}

func (d *notificationDisplay) UserChanged(_ context.Context, uid keybase1.UID) error {
	return d.printf("User %s changed\n", uid)
}