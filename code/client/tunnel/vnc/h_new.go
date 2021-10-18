package vnc

import (
	"fmt"
	"natpass/code/client/pool"
	"natpass/code/network"
	"net/http"
	"time"

	"github.com/lwch/logging"
	"github.com/lwch/runtime"
)

// New new vnc
func (v *VNC) New(pool *pool.Pool, w http.ResponseWriter, r *http.Request) {
	id, err := runtime.UUID(16, "0123456789abcdef")
	if err != nil {
		logging.Error("failed to generate link_id for vnc: %s, err=%v",
			v.Name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	conn := pool.Get(id)
	conn.SendConnectReq(id, v.cfg)
	link := NewLink(v, id, v.cfg.Target, conn)
	v.Lock()
	v.links[id] = link
	v.Unlock()
	ch := conn.ChanRead(id)
	timeout := time.After(conn.ReadTimeout)
	for {
		var msg *network.Msg
		select {
		case msg = <-ch:
		case <-timeout:
			logging.Error("create vnc %s on tunnel %s failed, timtout", link.id, link.parent.Name)
			http.Error(w, "timeout", http.StatusBadGateway)
			return
		}
		if msg.GetXType() != network.Msg_connect_rep {
			conn.Reset(id, msg)
			time.Sleep(conn.ReadTimeout / 10)
			continue
		}
		rep := msg.GetCrep()
		if !rep.GetOk() {
			logging.Error("create vnc %s on tunnel %s failed, err=%s",
				link.id, link.parent.Name, rep.GetMsg())
			http.Error(w, rep.GetMsg(), http.StatusBadGateway)
			return
		}
		break
	}
	logging.Info("new vnc: name=%s, id=%s", v.Name, id)
	fmt.Fprint(w, id)
}
