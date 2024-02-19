package main

type Response struct {
	Code int    `json:"Code"`
	Msg  string `json:"Message"`
	Data any    `json:"Data"`
}

// Response codes
const ResponseCodeSuccess = 0             // Success
const ResponseCodeSuccessLegacy = 1       // Success
const ResponseCodeParamsCheckError = 1001 // Parameter check error
const ResponseCodeServerError = 1002      // Server error
