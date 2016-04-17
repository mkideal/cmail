package server

const (
	stateNone = 1 << iota

	stateReady
	stateMailInput
	stateAuth
	stateExpectCmdEhlo
	stateExpectCmdAuth
	stateExpectCmdMail
	stateExpectCmdRcpt
	stateExpectCmdData
)
