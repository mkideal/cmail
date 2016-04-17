package server

const (
	stateNone = 1 << iota

	stateReady
	stateMailInput
	stateAuth
	stateExpectCmdAuth
	stateExpectCmdMail
	stateExpectCmdRcpt
	stateExpectCmdData
)
