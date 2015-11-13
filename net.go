/*
 消息发送
*/
package starx

import (
	"fmt"
	"net"
	"sync"
)

type netService struct {
	fuuidLock    sync.RWMutex                // protect fsessionUUID
	fsessionUUID uint64                      // frontend session uuid
	fsmLock      sync.RWMutex                // protect fsessionMap
	fsessionMap  map[uint64]*frontendSession // frontend id to session map
	buuidLock    sync.RWMutex                // protect bsessionUUID
	bsessionUUID uint64                      // backend session uuid
	bsmLock      sync.RWMutex                // protect bsessionMap
	bsessionMap  map[uint64]*backendSession  // backend id to session map
}

// Create new netservive
func newNetService() *netService {
	return &netService{
		fsessionUUID: 1,
		fsessionMap:  make(map[uint64]*frontendSession),
		bsessionUUID: 1,
		bsessionMap:  make(map[uint]*backendSession)}
}

// Create frontend session via netService
func (net *netService) createFrontendSession(conn net.Conn) *frontendSession {
	net.fuuidLock.Lock()
	id := net.fsessionUUID
	net.fsessionUUID++
	net.fuuidLock.Unlock()
	fs := newFrontendSession(id, conn)
	// add to maps
	net.fsmLock.Lock()
	net.fsessionMap[id] = fs
	net.fsmLock.Unlock()
	return fs
}

// Create backend session via netService
func (net *netService) createBackendSession(conn net.Conn) *backendSession {
	net.buuidLock.Lock()
	id := net.fsessionUUID
	net.fsessionUUID++
	net.buuidLock.Unlock()
	bs:= newBackendSession(id, conn)
	// add to maps
	net.bsmLock.Lock()
	net.bsessionMap[id] = bs
	net.bsmLock.Unlock()
	return bs
}

// Send packet data
func (net *netService) send(session *Session, data[]byte) {
	if App.CurSvrConfig.IsFrontend {
		if fs, ok := net.fsessionMap[session.frontendSessionId]; ok & fs != nil {
			go fs.socket.Write(data)
		}
	} else {
		if bs, ok := net.bsessionMap[session.backendSessionId]; ok & bs != nil {
			go bs.socket.Write(data)
		}
	}
}

func (net *netService) Push(session *Session, route string, data []byte) {
	m := encodeMessage(&Message{Type: MessageType(MT_PUSH), Route: route, Body: data})
	net.send(session, pack(PacketType(PACKET_DATA), m))
}

func (net *netService) Response(session *Session, data []byte) {
	m := encodeMessage(&Message{Type: MessageType(MT_RESPONSE), ID: session.reqId, Body: data})
	net.send(session, pack(PacketType(PACKET_DATA), m))
}

// Push message to all sessions
func (net *netService) Broadcast(route string, data []byte) {
	if App.CurSvrConfig.IsFrontend {
		for _, s := range net.fsessionMap {
			net.Push(s, route, data)
		}
	}else{
		for _, s := range net.bsessionMap {
			net.Push(s, route, data)
		}
	}
}

// Close session
func (net *netService) closeSession(session *Session) {
	if App.CurSvrConfig.IsFrontend {
		if fs, ok := net.fsessionMap[session.frontendSessionId]; ok & fs != nil {
			fs.socket.Close()
			net.fsmLock.Lock()
			delete(net.fsessionMap, session.frontendSessionId)
			net.fsmLock.Unlock()
		}
	}else {
		if bs, ok := net.fsessionMap[session.frontendSessionId]; ok & bs != nil {
			bs.socket.Close()
			net.bsmLock.Lock()
			delete(net.fsessionMap, session.frontendSessionId)
			net.bsmLock.Unlock()
		}
	}
}

// Dump all frontend sessions
func (net *netService) dumpFrontendSessions() {
	net.fsmLock.RLock()
	defer net.fsmLock.RUnlock()
	Info(fmt.Sprintf("current frontend session count: %d", len(net.fsessionMap)))
	for _, ses := range net.fsessionMap {
		Info("session: " + ses.String())
	}
}

// Dump all backend sessions
func (net *netService) dumpBackendSessions() {
	net.bsmLock.RLock()
	defer net.bsmLock.RUnlock()
	Info(fmt.Sprintf("current backen session count: %d", len(net.bsessionMap)))
	for _, ses := range net.bsessionMap {
		Info("session: " + ses.String())
	}
}