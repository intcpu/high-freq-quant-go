package notice

var (
	FeishuUrl    = "https://open.feishu.cn/open-apis/bot/hook/094b5926-e37f-4e84-b80a-5b4404ba4eb9"
	FeishuLogUrl = "https://open.feishu.cn/open-apis/bot/hook/de1ecd5f-048a-4302-99f3-98ab7f9cd834"
	ErrUrl       = "https://open.feishu.cn/open-apis/bot/v2/hook/094b5926-e37f-4e84-b80a-5b4404ba4eb9"
	LogUrl       = "https://open.feishu.cn/open-apis/bot/v2/hook/de1ecd5f-048a-4302-99f3-98ab7f9cd834"
)

//飞书推送接口
type FeishuMessage struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type Message struct {
	MsgType string  `json:"msg_type"`
	Content Content `json:"content"`
}

type Content struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type Response struct {
	Code          int         `json:"code"`
	Msg           string      `json:"msg"`
	Data          interface{} `json:"interface"`
	Extra         interface{} `json:"Extra"`
	StatusCode    int         `json:"StatusCode"`
	StatusMessage string      `json:"StatusMessage"`
}
