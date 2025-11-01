package transport

type CommonResp struct {
	Code int               `json:"code"`    // 0=OK, 非0=错误码
	Msg  CommonMessageResp `json:"message"` // 见下
}

// CommonMessageResp 既可返回数据，也可返回错误信息。
type CommonMessageResp struct {
	RequestID string      `json:"request_id,omitempty"` // 每次请求的追踪ID（便于排查）
	Data      interface{} `json:"data,omitempty"`       // 具体数据载荷（不同接口各自定义）
	Error     *ErrorInfo  `json:"error,omitempty"`      // 不为空则表示失败
}

// ErrorInfo 统一错误结构，便于统计与上游定位。
type ErrorInfo struct {
	Code     string                 `json:"code"`               // 业务/上游错误码，如 "HF_429" / "DEESEEK_500"
	Message  string                 `json:"message"`            // 人类可读的错误文案
	Upstream string                 `json:"upstream,omitempty"` // 上游来源：hf/deepseek/vectorize/gateway
	Details  map[string]interface{} `json:"details,omitempty"`  // 额外上下文（如 status, retryAfter 等）
}

func OK(data interface{}) CommonResp { return CommonResp{Code: 0, Msg: CommonMessageResp{Data: data}} }
func Err(e *ErrorInfo) CommonResp    { return CommonResp{Code: 1, Msg: CommonMessageResp{Error: e}} }
