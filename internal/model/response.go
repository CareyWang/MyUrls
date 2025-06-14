package model

type Response struct {
	Code int    `json:"Code"`
	Msg  string `json:"Message"`
	Data any    `json:"Data"`
}

// Response codes
const (
	ResponseCodeSuccess          = 0    // Success
	ResponseCodeSuccessLegacy    = 1    // Success
	ResponseCodeParamsCheckError = 1001 // Parameter check error
	ResponseCodeServerError      = 1002 // Server error
)
