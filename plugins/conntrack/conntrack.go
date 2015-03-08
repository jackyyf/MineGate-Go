package conntrack

import (
	"github.com/deckarep/golang-set"
	"github.com/jackyyf/MineGate-Go/minegate"
	log "github.com/jackyyf/golog"
	"strings"
	"sync"
	"time"
)

var brust uint64 = 5
var limit uint64 = 5
var delta uint64 = 60

var ollock sync.Mutex

type loginInfo struct {
	User   string
	Server string
}

// Track online user with upstream
var conn_in = make(map[uint64]loginInfo)

// Track online user for each server.
var online_list = make(map[string]mapset.Set)

// Track connection per ip
var ctlock sync.Mutex
var conn_track = make(map[string]uint64)

func init() {
	minegate.OnPostLoadConfig(loadConfig, 0)
	minegate.OnPostAccept(connectionHandler, 39)
	minegate.OnLoginRequest(userLogin, 39)
	minegate.OnDisconnect(userLogout, 39)
	go heartBeatTicker()
}

func heartBeatTicker() {
	for {
		time.Sleep(time.Second * time.Duration(delta))
		if limit == 0 {
			continue
		}
		ctlock.Lock()
		log.Infof("[conntrack] Reducing connection count...")
		for track := range conn_track {
			if conn_track[track] <= limit {
				delete(conn_track, track)
				log.Debugf("[conntrack] Removing IP %s.", track)
			} else {
				conn_track[track] -= limit
			}

		}
		log.Infof("[conntrack] Done. IPs currently in record: %d", len(conn_track))
		ctlock.Unlock()
	}
}

func loadConfig() {
	new_brust, err := minegate.GetExtraConf("conntrack.brust")
	if err != nil {
		log.Info("[conntrack] Using default brust: 5")
		brust = 5
	} else {
		nb := minegate.ToUint(new_brust)
		log.Infof("[conntrack] Using brust: %d", nb)
		brust = nb
	}
	new_limit, err := minegate.GetExtraConf("conntrack.limit")
	if err != nil {
		log.Info("[conntrack] Using default limit: 5")
		limit = 5
	} else {
		nl := minegate.ToUint(new_limit)
		if nl != 0 {
			log.Infof("[conntrack] Using limit: %d", nl)
			limit = nl
		} else {
			log.Info("[conntrack] Limit = 0, disabling conntrack.")
			limit = 0
		}
	}
	new_delta, err := minegate.GetExtraConf("conntrack.interval")
	if err != nil {
		log.Info("[conntrack] Using default delta: 60")
		delta = 60
	} else {
		nd := minegate.ToUint(new_delta)
		if nd < 15 {
			log.Warn("[conntrack] Minimal interval is 15.")
			nd = 15
		}
		log.Infof("[conntrack] Using interval %d", nd)
		delta = nd
	}
}

func connectionHandler(event *minegate.PostAcceptEvent) {
	if event.Rejected() {
		return
	}
	ip := event.RemoteAddr.IP.String()
	if limit == 0 {
		return
	}
	ctlock.Lock()
	defer ctlock.Unlock()
	if conn_track[ip] >= brust+limit {
		event.Warnf("[conntrack] Rejected: reach connection limit.")
		event.Reject()
		return
	}
	conn_track[ip] += 1
	event.Infof("[conntrack] Accepted: connection count = %d", conn_track[ip])
}

func userLogin(event *minegate.LoginRequestEvent) {
	if event.Rejected() {
		return
	}
	server := strings.ToLower(event.Upstream.Server)
	user := strings.ToLower(event.LoginPacket.Name)
	connID := event.GetConnID()
	ollock.Lock()
	defer ollock.Unlock()
	if online_list[server] == nil {
		online_list[server] = mapset.NewThreadUnsafeSet()
	}
	s := online_list[server]
	if !s.Add(user) {
		event.Warnf("[conntrack] User %s is already in server %s, rejected.", event.LoginPacket.Name, event.Upstream.Server)
		event.Reason("You are already in this server.")
		return
	}
	conn_in[connID] = loginInfo{
		Server: server,
		User:   user,
	}
	event.Infof("[conntrack] User %s joined server %s.", event.LoginPacket.Name, event.Upstream.Server)
	event.Infof("[conntrack] Upstream online user: %d. Total online user: %d.", online_list[server].Cardinality(), len(conn_in))
}

func userLogout(event *minegate.DisconnectEvent) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("[conntrack] Recovered from panic. Programming error inside plugin. Please contact author ASAP!")
		}
	}()
	connID := event.GetConnID()
	ollock.Lock()
	defer ollock.Unlock()
	info, ok := conn_in[connID]
	if !ok {
		return
	}
	delete(conn_in, connID)
	online_list[info.Server].Remove(info.User)
	event.Infof("[conntrack] User %s logout from server %s.", info.User, info.Server)
	event.Infof("[conntrack] Upstream online user: %d. Total online user: %d", online_list[info.Server].Cardinality(), len(conn_in))
}
