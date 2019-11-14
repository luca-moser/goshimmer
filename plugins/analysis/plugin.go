package analysis

import (
	"github.com/iotaledger/goshimmer/packages/daemon"
	"github.com/iotaledger/goshimmer/packages/node"
	"github.com/iotaledger/goshimmer/plugins/analysis/client"
	"github.com/iotaledger/goshimmer/plugins/analysis/server"
	"github.com/iotaledger/goshimmer/plugins/analysis/webinterface"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/parameter"
)

var PLUGIN = node.NewPlugin("Analysis", node.Enabled, configure, run)

func configure(plugin *node.Plugin) {
	if parameter.NodeConfig.GetInt(server.CFG_SERVER_PORT) != 0 {
		webinterface.Configure(plugin)
		server.Configure(plugin)

		daemon.Events.Shutdown.Attach(events.NewClosure(func() {
			server.Shutdown(plugin)
		}))
	}
}

func run(plugin *node.Plugin) {
	if parameter.NodeConfig.GetInt(server.CFG_SERVER_PORT) != 0 {
		webinterface.Run(plugin)
		server.Run(plugin)
	} else {
		plugin.Node.LogSuccess("Node", "Starting Plugin: Analysis ... server is disabled (server-port is 0)")
	}

	if parameter.NodeConfig.GetString(client.CFG_SERVER_ADDRESS) != "" {
		client.Run(plugin)
	} else {
		plugin.Node.LogSuccess("Node", "Starting Plugin: Analysis ... client is disabled (server-address is empty)")
	}
}
