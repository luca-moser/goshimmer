package dashboard

import (
	"encoding/binary"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/iotaledger/goshimmer/packages/accountability"
	"github.com/iotaledger/goshimmer/plugins/autopeering/instances/acceptedneighbors"
	"github.com/iotaledger/goshimmer/plugins/autopeering/instances/chosenneighbors"
	"github.com/iotaledger/goshimmer/plugins/autopeering/instances/knownpeers"
	"github.com/iotaledger/goshimmer/plugins/autopeering/instances/neighborhood"
	"github.com/iotaledger/goshimmer/plugins/metrics"
	"github.com/iotaledger/hive.go/events"
)

var (
	start                = time.Now()
	homeTempl, templ_err = template.New("dashboard").Parse(tpsTemplate)
	upgrader             = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Status struct {
	Id        string `json:"Id"`
	Neighbor  string `json:"Neighbor"`
	KnownPeer string `json:"KnownPeer"`
	Uptime    string `json:"Uptime"`
}

func GetStatus() *Status {
	// Get Uptime
	duration := time.Since(start)
	padded := false
	uptime := fmt.Sprintf("Uptime: ")
	if int(duration.Seconds())/(60*60*24) > 0 {
		days := int(duration.Hours()) / 24

		numberLength := int(math.Log10(float64(days))) + 1
		padLength := 31 - numberLength

		uptime += fmt.Sprintf("%*v", padLength, "")
		uptime += fmt.Sprintf("%02dd ", days)
	}

	if int(duration.Seconds())/(60*60) > 0 {
		if !padded {
			uptime += fmt.Sprintf("%29v", "")
			padded = true
		}
		uptime += fmt.Sprintf("%02dh ", int(duration.Hours())%24)
	}

	if int(duration.Seconds())/60 > 0 {
		if !padded {
			uptime += fmt.Sprintf("%33v", "")
			padded = true
		}
		uptime += fmt.Sprintf("%02dm ", int(duration.Minutes())%60)
	}

	if !padded {
		uptime += fmt.Sprintf("%37v", "")
	}
	uptime += fmt.Sprintf("%02ds  ", int(duration.Seconds())%60)

    return &Status {
	Id: accountability.OwnId().StringIdentifier,
	Neighbor: "Neighbors:  " + strconv.Itoa(chosenneighbors.INSTANCE.Peers.Len())+" chosen / "+strconv.Itoa(acceptedneighbors.INSTANCE.Peers.Len())+" accepted / "+strconv.Itoa(chosenneighbors.INSTANCE.Peers.Len()+acceptedneighbors.INSTANCE.Peers.Len())+" total",
	KnownPeer: "Known Peers: "+ strconv.Itoa(knownpeers.INSTANCE.Peers.Len())+" total / "+strconv.Itoa(neighborhood.INSTANCE.Peers.Len())+" neighborhood",
	Uptime: uptime,
    }
}

// ServeWs websocket
func ServeWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	var websocketWriteMutex sync.Mutex

	notifyWebsocketClient := events.NewClosure(func(sampledTPS uint64) {
		go func() {
			websocketWriteMutex.Lock()
			defer websocketWriteMutex.Unlock()

			p := make([]byte, 4)
			binary.LittleEndian.PutUint32(p, uint32(sampledTPS))
			if err := ws.WriteMessage(websocket.BinaryMessage, p); err != nil {
				return
			}

			// write node status message
			status := GetStatus()
			if err := ws.WriteJSON(status); err != nil {
				return
			}
		}()
	})

	metrics.Events.ReceivedTPSUpdated.Attach(notifyWebsocketClient)

	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			break
		}
	}

	metrics.Events.ReceivedTPSUpdated.Detach(notifyWebsocketClient)
}

// ServeHome registration
func ServeHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/dashboard" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if templ_err != nil {
		panic("HTML template error")
	}

	_ = homeTempl.Execute(w, &struct {
		Host string
		Data []uint32
	}{
		r.Host,
		TPSQ,
	})
}

var TPSQ []uint32

const (
	MAX_Q_SIZE int = 3600
)
