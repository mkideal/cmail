package server

const (
	CodeSystemStatus                        = 211
	CodeHelpMessage                         = 214
	CodeServiceReady                        = 220
	CodeServiceClosing                      = 221
	CodeOK                                  = 250
	CodeUserNotLocal                        = 251
	CodeCannotVRFYUser                      = 252
	CodeStartMailInput                      = 354
	CodeServiceNotAvailable                 = 421
	CodeMailboxUnavailable                  = 450
	CodeLocalErrorInProcessing              = 451
	CodeInsufficientSystemStorage           = 452
	CodeServerUnableToAccommodateParameters = 455
	CodeSyntaxError                         = 500
	CodeSyntaxErrorInParametersOrArguments  = 501
	CodePermCommandNotImplemented           = 502
	CodePermBadSequenceOfCommands           = 503
	CodePermCommandParameterNotImplemented  = 504
	CodePermMailboxUnavailable              = 550
	CodePermUserNotLocal                    = 551
	CodePermExceededStorageAllocation       = 552
	CodePermMailboxNameNotAllowed           = 553
	CodePermTransactionFailed               = 554
	CodePermMailRcptParameterError          = 555
)
