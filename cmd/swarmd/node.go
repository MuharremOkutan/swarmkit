package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"

	"github.com/Sirupsen/logrus"
	engineapi "github.com/docker/engine-api/client"
	"github.com/docker/swarm-v2/agent"
	"github.com/docker/swarm-v2/agent/exec/container"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Run the swarm node",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		hostname, err := cmd.Flags().GetString("hostname")
		if err != nil {
			return err
		}
		addr, err := cmd.Flags().GetString("listen-remote-api")
		if err != nil {
			return err
		}
		addrHost, _, err := net.SplitHostPort(addr)
		if err == nil {
			ip := net.ParseIP(addrHost)
			if ip != nil && (ip.IsUnspecified() || ip.IsLoopback()) {
				fmt.Println("Warning: Specifying a valid address with --listen-remote-api may be necessary for other managers to reach this one.")
			}
		}

		unix, err := cmd.Flags().GetString("listen-control-api")
		if err != nil {
			return err
		}

		managerAddr, err := cmd.Flags().GetString("join-addr")
		if err != nil {
			return err
		}

		forceNewCluster, err := cmd.Flags().GetBool("force-new-cluster")
		if err != nil {
			return err
		}

		hb, err := cmd.Flags().GetUint32("heartbeat-tick")
		if err != nil {
			return err
		}

		election, err := cmd.Flags().GetUint32("election-tick")
		if err != nil {
			return err
		}

		stateDir, err := cmd.Flags().GetString("state-dir")
		if err != nil {
			return err
		}

		caHash, err := cmd.Flags().GetString("ca-hash")
		if err != nil {
			return err
		}

		secret, err := cmd.Flags().GetString("secret")
		if err != nil {
			return err
		}

		engineAddr, err := cmd.Flags().GetString("engine-addr")
		if err != nil {
			return err
		}

		// todo: temporary to bypass promotion not working yet
		ismanager, err := cmd.Flags().GetBool("manager")
		if err != nil {
			return err
		}

		// Create a context for our GRPC call
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, err := engineapi.NewClient(engineAddr, "", nil, nil)
		if err != nil {
			return err
		}

		executor := container.NewExecutor(client)

		n, err := agent.NewNode(&agent.NodeConfig{
			Hostname:         hostname,
			ForceNewCluster:  forceNewCluster,
			ListenControlAPI: unix,
			ListenRemoteAPI:  addr,
			JoinAddr:         managerAddr,
			StateDir:         stateDir,
			CAHash:           caHash,
			Secret:           secret,
			Executor:         executor,
			HeartbeatTick:    hb,
			ElectionTick:     election,
			IsManager:        ismanager,
		})
		if err != nil {
			return err
		}

		if err := n.Start(ctx); err != nil {
			return err
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			<-c
			n.Stop(ctx)
		}()

		go func() {
			<-n.Ready(ctx)
			if ctx.Err() == nil {
				logrus.Info("node is ready")
			}
		}()

		return n.Err(context.Background())
	},
}

func init() {
	nodeCmd.Flags().String("engine-addr", "unix:///var/run/docker.sock", "Address of engine instance of agent.")
	nodeCmd.Flags().String("hostname", "", "Override reported agent hostname")
	nodeCmd.Flags().String("listen-remote-api", "0.0.0.0:4242", "Listen address for remote API")
	nodeCmd.Flags().String("listen-control-api", "/var/run/docker/cluster/docker-swarmd.sock", "Listen socket for control API")
	nodeCmd.Flags().String("join-addr", "", "Join cluster with a node at this address")
	nodeCmd.Flags().Bool("force-new-cluster", false, "Force the creation of a new cluster from data directory")
	nodeCmd.Flags().Uint32("heartbeat-tick", 1, "Defines the heartbeat interval (in seconds) for raft member health-check")
	nodeCmd.Flags().Uint32("election-tick", 3, "Defines the amount of ticks (in seconds) needed without a Leader to trigger a new election")
	nodeCmd.Flags().Bool("manager", false, "Request initial CSR in a manager role")
}