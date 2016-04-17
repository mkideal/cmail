package server

import (
	"net"
	"net/mail"
	"sync"
	"sync/atomic"

	"github.com/mkideal/smtpd/etc"
)

// Repository represents email repository
type Repository interface {
	FindMailbox(usernameOrAddress string) (*mail.Address, bool)
	SaveEmail(from, tos string, data []byte) error
}

//--------
// Server
//--------

type Server struct {
	repo Repository

	locker       sync.Mutex
	sessions     map[uint64]*session
	curSessionId uint64
}

func New(repo Repository) *Server {
	svr := new(Server)
	svr.repo = repo
	svr.sessions = make(map[uint64]*session)
	return svr
}

func (svr *Server) Start(addr string, onListenErr, onAcceptErr func(error)) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		onListenErr(err)
		return
	}
	onListenErr(nil)
	for {
		c, err := listener.Accept()
		if err != nil {
			onAcceptErr(err)
			return
		}
		s := newSession(svr, c)
		s.id = svr.allocSessionId()
		if svr.addSession(s) {
			//TODO: 超时退出session
			go s.run()
		}
	}
}

func (svr *Server) allocSessionId() uint64 {
	return atomic.AddUint64(&svr.curSessionId, 1)
}

func (svr *Server) addSession(s *session) bool {
	svr.locker.Lock()
	defer svr.locker.Unlock()
	if len(svr.sessions) >= etc.Conf().MaxSessionSize {
		return false
	}
	svr.sessions[s.id] = s
	return true
}

func (svr *Server) removeSession(sid uint64) {
	svr.locker.Lock()
	defer svr.locker.Unlock()
	delete(svr.sessions, sid)
}
