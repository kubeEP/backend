package constant

type ResponseStatus string

const (
	Fail    ResponseStatus = "fail"
	Success ResponseStatus = "success"
	Error   ResponseStatus = "error"
)
