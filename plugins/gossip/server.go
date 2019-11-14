package gossip

import (
	"github.com/iotaledger/goshimmer/packages/accountability"
	"github.com/iotaledger/goshimmer/packages/daemon"
	"github.com/iotaledger/goshimmer/packages/errors"
	"github.com/iotaledger/goshimmer/packages/identity"
	"github.com/iotaledger/goshimmer/packages/network"
	"github.com/iotaledger/goshimmer/packages/network/tcp"
	"github.com/iotaledger/goshimmer/packages/node"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/parameter"
)

var TCPServer = tcp.NewServer()

func configureServer(plugin *node.Plugin) {
	TCPServer.Events.Connect.Attach(events.NewClosure(func(conn *network.ManagedConnection) {
		protocol := newProtocol(conn)

		// print protocol errors
		protocol.Events.Error.Attach(events.NewClosure(func(err errors.IdentifiableError) {
			plugin.LogFailure(err.Error())
		}))

		// store protocol in neighbor if its a neighbor calling
		protocol.Events.ReceiveIdentification.Attach(events.NewClosure(func(identity *identity.Identity) {
			if protocol.Neighbor != nil {

				if protocol.Neighbor.GetAcceptedProtocol() == nil {
					protocol.Neighbor.SetAcceptedProtocol(protocol)

					protocol.Conn.Events.Close.Attach(events.NewClosure(func() {
						protocol.Neighbor.SetAcceptedProtocol(nil)
					}))
				}
			}
		}))

		// drop the "secondary" connection upon successful handshake
		protocol.Events.HandshakeCompleted.Attach(events.NewClosure(func() {
			if protocol.Neighbor.GetIdentity().StringIdentifier <= accountability.OwnId().StringIdentifier {
				var initiatedProtocolConn *network.ManagedConnection
				if protocol.Neighbor.GetInitiatedProtocol() != nil {
					initiatedProtocolConn = protocol.Neighbor.GetInitiatedProtocol().Conn
				}

				if initiatedProtocolConn != nil {
					_ = initiatedProtocolConn.Close()
				}
			}

			protocol.Neighbor.Events.ProtocolConnectionEstablished.Trigger(protocol)
		}))

		go protocol.Init()
	}))

	daemon.Events.Shutdown.Attach(events.NewClosure(func() {
		plugin.LogInfo("Stopping TCP Server ...")

		TCPServer.Shutdown()
	}))
}

func runServer(plugin *node.Plugin) {
	gossipPort := parameter.NodeConfig.GetString(GOSSIP_PORT)
	plugin.LogInfo("Starting TCP Server (port " + gossipPort + ") ...")

	daemon.BackgroundWorker("Gossip TCP Server", func() {
		plugin.LogSuccess("Starting TCP Server (port " + gossipPort + ") ... done")

		TCPServer.Listen(parameter.NodeConfig.GetInt(GOSSIP_PORT))

		plugin.LogSuccess("Stopping TCP Server ... done")
	})
}
