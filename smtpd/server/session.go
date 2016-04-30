package server

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	//"net/smtp"
	"net/textproto"
	"regexp"
	"sort"
	"strings"

	"github.com/mkideal/cmail/smtpd/etc"
	"github.com/mkideal/pkg/debug"
)

var (
	fromRegexp = regexp.MustCompile("[Ff][Rr][Oo][Mm]:(.+)")
	toRegexp   = regexp.MustCompile("[Tt][Oo]:(.+)")
)

const crlf = "\r\n"

//---------
// command
//---------

const (
	NONE = ""

	HELP     = "HELP"
	VRFY     = "VRFY"
	EXPN     = "EXPN"
	SIZE     = "SIZE"
	STARTTLS = "STARTTLS"

	EHLO = "EHLO"
	HELO = "HELO"
	MAIL = "MAIL"
	RCPT = "RCPT"
	DATA = "DATA"

	AUTH          = "AUTH"
	RSET          = "RSET"
	NOOP          = "NOOP"
	QUIT          = "QUIT"
	EIGHT_BITMIME = "8BITMIME"
)

type command struct {
	isExt bool
	state int
}

var commands = map[string]command{
	HELP:     command{true, stateNone},
	VRFY:     command{true, stateNone},
	EXPN:     command{true, stateNone},
	SIZE:     command{true, stateNone},
	STARTTLS: command{true, stateNone},
	AUTH:     command{true, stateExpectCmdAuth},

	HELO: command{false, stateNone},
	EHLO: command{false, stateNone},
	MAIL: command{false, stateExpectCmdMail},
	RCPT: command{false, stateExpectCmdRcpt},
	DATA: command{false, stateExpectCmdData},

	RSET: command{false, stateNone},
	NOOP: command{false, stateNone},
	QUIT: command{false, stateNone},
}

// supported extensions
var ext = func() []string {
	s := []string{}
	for name, cmd := range commands {
		if cmd.isExt {
			s = append(s, name)
		}
	}
	sort.Strings(s)
	return s
}()

var extString = func() string {
	buf := bytes.NewBufferString("")
	for i, e := range ext {
		if i+1 == len(ext) {
			buf.WriteString("250 ")
		} else {
			buf.WriteString("250-")
		}
		buf.WriteString(e)
		buf.WriteString(crlf)
	}
	return buf.String()
}()

//---------
// session
//---------

type session struct {
	id         uint64
	svr        *Server
	nativeConn net.Conn
	conn       *textproto.Conn

	// whether the session is using TLS
	tls bool

	// auth buffer
	auth []byte

	// reverse-path buffer
	from *mail.Address

	// forward-path buffer
	tos []*mail.Address

	// data buffer
	data *bytes.Buffer

	// current state
	state int

	// error counter
	errCount int
}

func newSession(svr *Server, conn net.Conn) *session {
	s := new(session)
	s.svr = svr
	s.nativeConn = conn
	s.conn = textproto.NewConn(conn)
	s.state = stateReady

	// init buffer
	s.auth = []byte{}
	s.data = bytes.NewBufferString("")
	s.tos = []*mail.Address{}

	return s
}

func (s *session) setState(state int) {
	s.state = state
	debug.Debugf("session %d switch to state %x", s.id, state)
}

func (s *session) quit() {
	s.svr.removeSession(s.id)
	s.conn.Close()
}

func (s *session) run() {
	s.conn.PrintfLine("%3d %s", CodeServiceReady, etc.Conf().S_ServiceInfo)
	for {
		if s.errCount >= etc.Conf().MaxErrorSize {
			s.quit()
			return
		}
		line, err := s.conn.ReadLine()
		if err != nil {
			debug.Debugf("session %d read error: %v", s.id, err)
			s.quit()
			return
		}

		var (
			quit  bool
			isCmd bool
		)

		debug.Debugf("session %d state: %x", s.id, s.state)
		switch s.state {
		case stateMailInput:
			quit = s.appendData(line)

		case stateAuth:
			//quit = s.auth(line)

		default:
			isCmd = true
		}

		if quit {
			s.quit()
			return
		}
		if !isCmd {
			continue
		}

		var (
			cmdName = ""
			args    = ""
			strs    = strings.SplitN(line, " ", 2)
		)
		if len(strs) > 0 {
			cmdName = strs[0]
		}
		if len(strs) > 1 {
			args = strs[1]
		}
		if quit := s.dispatch(cmdName, args); quit {
			s.quit()
			return
		}
	}
}

func (s *session) isExpectedCmd(cmd command) bool {
	return cmd.state == stateNone || (s.state&cmd.state) != 0
}

func (s *session) dispatch(cmdName, args string) (quit bool) {
	if cmdName == "" {
		return
	}
	debug.Debugf("session %d recv command: %q, args: %q", s.id, cmdName, args)
	cmd, ok := commands[cmdName]
	if !ok {
		s.commandNotImplemented(cmdName)
		return
	}
	if !s.isExpectedCmd(cmd) {
		s.responseBadSequence()
		return
	}
	switch cmdName {
	case NOOP:
		s.onNoOp()

	case HELP:
		s.onHelp(args)

	case EXPN:
		s.commandNotImplemented(cmdName)

	case VRFY:
		s.onVrfy(args)

	case RSET:
		s.onRset(args)

	case STARTTLS:
		quit = s.onStartTLS(args)

	case QUIT:
		quit = s.onQuit(args)

	case HELO:
		s.onHelo(args)
	case EHLO:
		s.onEhlo(args)

	case AUTH:
		quit = s.onAuth(args)

	case MAIL:
		quit = s.onMail(args)

	case RCPT:
		quit = s.onRcpt(args)

	case DATA:
		quit = s.onData(args)

	default:
		s.commandNotImplemented(cmdName)
	}
	return quit
}

func (s *session) commandNotImplemented(cmd string) {
	s.responseCommandNotImplemented(cmd)
}

func (s *session) appendData(args string) (quit bool) {
	if args == "." {
		return s.complete()
	}
	if s.data.Len()+len(args) > etc.Conf().MaxBufferSize {
		s.responseExceededStorage()
		return
	}
	s.data.WriteString(args)
	s.data.WriteString(crlf)
	s.responseOK()
	return
}

func (s *session) complete() (quit bool) {
	if s.from == nil || s.tos == nil || len(s.tos) == 0 {
		s.responseBadSequence()
		return
	}
	buf := bytes.NewBufferString("")
	for i, to := range s.tos {
		if i != 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(to.String())
	}

	var (
		fromAddrStr  = s.from.String()
		toAddrStr    = buf.String()
		mailData     = s.data.Bytes()
		serverDomain = etc.Conf().DomainName
		allowDelay   = etc.Conf().AllowDelay
	)

	for _, to := range s.tos {
		toDomain := parseDomainFromAddress(to.Address)
		if toDomain != serverDomain {
			// delay mail
			fromDomain := parseDomainFromAddress(s.from.Address)
			if fromDomain != serverDomain && !aAllowDelay {
				//TODO: handle the error
				debug.Debugf("cannot delay mail")
			} else {
				debug.Debugf("delay mail ...")
				delayMail(toDomain, fromAddrStr, to.Address, mailData)
			}
			continue
		}

		err := s.svr.repo.SaveEmail(to, fromAddrStr, toAddrStr, mailData)
		if err != nil {
			s.responseLocalError()
			return
		}
	}
	s.responseOK()
	return
}

func parseDomainFromAddress(address string) string {
	index := strings.Index(address, "@")
	if index >= 0 {
		return address[index+1:]
	}
	return etc.Conf().DomainName
}

//------------------
// command handlers
//------------------

// HELP
func (s *session) onHelp(args string) {
	s.printf("%3d https://tools.ietf.org/html/rfc5321", CodeHelpMessage)
}

// HELO
func (s *session) onHelo(args string) {
	if len(args) > 0 {
		s.responseOK()
		s.setState(stateExpectCmdMail | stateExpectCmdAuth)
	} else {
		s.responseSyntaxError()
	}
}

// EHLO
func (s *session) onEhlo(args string) {
	if len(args) > 0 {
		s.printf(extString)
		s.setState(stateExpectCmdMail | stateExpectCmdAuth)
	} else {
		s.responseSyntaxError()
	}
}

// NOOP
func (s *session) onNoOp() {
	s.responseOK()
}

// VRFY
func (s *session) onVrfy(args string) {
	name := args
	if addr, err := mail.ParseAddress(args); err == nil {
		if addr.Address != "" {
			name = addr.Address
		}
	}
	addr, ok := s.svr.repo.FindMailbox(name)
	if ok {
		s.responseVrfy(addr.String())
		return
	}
	s.responseUserNotLocal()
}

// RSET
// RFC 4.1.1.5:
// "This command specifies that the current mail transaction will be
// aborted.  Any stored sender, recipients, and mail data MUST be
// discarded, and all buffers and state tables cleared.  The receiver
// MUST send a "250 OK" reply to a RSET command with no arguments."
func (s *session) onRset(args string) {
	if args != "" {
		s.responseErrorInParameter()
		return
	}
	s.reset()
	s.responseOK()
}

func (s *session) reset() {
	s.from = nil
	s.tos = s.tos[0:0]
	s.auth = s.auth[0:0]
	s.data.Reset()
	s.setState(stateExpectCmdMail | stateExpectCmdAuth)
}

// STARTTLS
func (s *session) onStartTLS(args string) (quit bool) {
	if s.svr.tlsConfig == nil {
		s.commandNotImplemented(STARTTLS)
		return
	}
	tlsConn := tls.Server(s.nativeConn, s.svr.tlsConfig)
	s.nativeConn = tlsConn
	s.conn = textproto.NewConn(tlsConn)
	s.tls = true
	s.reset()
	s.responseOK()
	return
}

// AUTH
func (s *session) onAuth(args string) (quit bool) {
	s.commandNotImplemented(AUTH)
	return
}

// MAIL
// RFC5321 4.1.1.2:
// "This command clears the reverse-path buffer, the forward-path buffer,
// and the mail data buffer, and it inserts the reverse-path information
// from its argument clause into the reverse-path buffer."
func (s *session) onMail(args string) (quit bool) {
	matchResult := fromRegexp.FindStringSubmatch(args)
	if matchResult == nil || len(matchResult) != 2 {
		s.responsePermMailRcptParameterError()
		return
	}
	if addr, err := mail.ParseAddress(matchResult[1]); err != nil {
		s.responsePermMailRcptParameterError()
	} else {
		s.from = addr
		s.tos = s.tos[0:0]
		s.data.Reset()
		s.setState(stateExpectCmdRcpt)
		s.responseOK()
	}
	return
}

// RCPT
// RFC5321 4.1.1.3:
// "This command appends its forward-path argument to the forward-path
// buffer; it does not change the reverse-path buffer nor the mail data
// buffer."
func (s *session) onRcpt(args string) (quit bool) {
	matchResult := toRegexp.FindStringSubmatch(args)
	if matchResult == nil || len(matchResult) != 2 {
		s.responsePermMailRcptParameterError()
		return
	}
	addr, err := mail.ParseAddress(matchResult[1])
	if err != nil {
		s.responsePermMailRcptParameterError()
		return
	}
	if len(s.tos) >= etc.Conf().MaxRecipients {
		s.responseTooManyRecipients()
		return
	}
	s.responseOK()
	s.tos = append(s.tos, addr)
	s.setState(stateExpectCmdData | stateExpectCmdRcpt)
	return
}

// DATA
func (s *session) onData(args string) (quit bool) {
	if args != "" {
		s.responseErrorInParameter()
		return
	}
	s.responseStartMailInput()
	s.setState(stateMailInput)
	return
}

// QUIT
func (s *session) onQuit(args string) bool {
	if args != "" {
		s.responseErrorInParameter()
		return false
	}
	s.responseQuit()
	return true
}

//----------
// response
//----------

func (s *session) responseOK() {
	s.printf("%3d OK", CodeOK)
}

func (s *session) responseQuit() {
	s.printf("%3d bye", CodeServiceClosing)
}

func (s *session) responsePermMailRcptParameterError() {
	s.errCount++
	s.printf("%3d mail/rcpt parameter syntax error", CodePermMailRcptParameterError)
}

func (s *session) responseSyntaxError() {
	s.errCount++
	s.printf("%3d syntax error", CodeSyntaxError)
}

func (s *session) responseErrorInParameter() {
	s.errCount++
	s.printf("%3d syntax error", CodeSyntaxErrorInParametersOrArguments)
}

func (s *session) responseCommandNotImplemented(cmd string) {
	s.errCount++
	s.printf("%3d command %q not implemented", CodePermCommandNotImplemented, cmd)
}

func (s *session) responseBadSequence() {
	s.errCount++
	s.printf("%3d bad sequence of commands", CodePermBadSequenceOfCommands)
}

func (s *session) responseStartMailInput() {
	s.printf("%3d start mail input", CodeStartMailInput)
}

func (s *session) responseVrfy(addr string) {
	s.printf("%3d %s", CodeOK, addr)
}

func (s *session) responseUserNotLocal() {
	s.printf("%3d user not local", CodeUserNotLocal)
}

func (s *session) responseTooManyRecipients() {
	s.errCount++
	s.printf("%3d too many recipients", CodeInsufficientSystemStorage)
}

func (s *session) responseExceededStorage() {
	s.errCount++
	s.printf("%3d exceeded storage", CodePermExceededStorageAllocation)
}

func (s *session) responseLocalError() {
	s.errCount++
	s.printf("%3d save email error", CodeLocalErrorInProcessing)
}

func (s *session) printf(format string, args ...interface{}) {
	resp := fmt.Sprintf(format, args...)
	debug.Debugf("resp: %s", resp)
	s.conn.PrintfLine(resp)
}
