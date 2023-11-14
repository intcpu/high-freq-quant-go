package exch

type PubChan chan interface{}

type BaseInfo struct {
	Symbol           string  //交易对
	Base             string  //交易货币
	Quote            string  //计价货币
	Unit             float64 //交易币整数单位
	MinBase          float64 //最小交易币数量
	MinQuote         float64 //最小计价币数量
	PriceFloat       int     //价格挂单精度
	SizeFloat        int     //数量挂单精度
	MinPriceStep     float64 //价格最小挂单单位
	MinSizeStep      float64 //数量最小挂单单位
	MaxOrderPrice    float64 //最大挂单价格
	MinOrderPrice    float64 //最小挂单价格
	MarkPrice        float64 //标记价
	IndexPrice       float64 //指数价
	FundingRate      float64 //当前资金汇率
	FundingNextApply float64 //下次资金汇率时间
	ChangeRate       float64 //涨跌幅
	DayVolume        float64 //一天成交额
	TakerFeeRate     float64 //taker 手续费
	MakerFeeRate     float64 //maker 手续费
	ExpireTime       int64   //交割时间
	LastUpdateTime   int64   //最后更新时间
}

type Balance struct {
	ApiSign string  //api账号
	Asset   string  //资产名称
	Total   float64 //账号总额
	Avative float64 //合约可用
}

type Position struct {
	Symbol         string  //交易对
	Size           float64 //开仓数量
	Price          float64 //开仓均价
	Margin         float64 //仓位保证金
	Pnl            float64 //已实现盈亏
	UnPnl          float64 //未实现盈亏
	LiqPrice       float64 //爆仓价
	MarkPrice      float64 //标记价
	Lv             float64 //杠杠
	MarginType     string  //保证金类型 逐仓、全仓
	PositionMode   string  //持仓模式 单仓、多仓
	OpenMargin     float64 //开仓保证金
	Value          float64 //仓位价值
	LastUpdateTime int64   //最后更新时间
}
