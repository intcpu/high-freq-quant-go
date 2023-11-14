package exch

const (
	OrderMaker = "maker"
	OrderTaker = "taker"

	OrderOpen     = "open"
	OrderFinished = "finished"

	OrderGtc = "gtc" //成交为止
	OrderIoc = "ioc" //立即成交或者取消，只吃单不挂单
	OrderPoc = "poc" //被动委托，只挂单不吃单
	OrderFok = "fok" //无法全部立即成交就撤销

	OrderLimit  = "limit"
	OrderMarket = "market"
	OrderStop   = "stop"

	MarginIsolated = "isolated" //保证金逐仓
	MarginCrossed  = "crossed"  //保证金全仓

	PositionBoth  = "BOTH"  //单仓
	PositionLong  = "LONG"  //全仓多仓
	PositionShort = "SHORT" //全仓空仓
)

type Order struct {
	ApiSign    string  //api
	Id         string  //订单ID
	TradeId    string  //成交单ID
	UUID       string  //自定义订单Id
	Symbol     string  //交易对
	Status     string  //状态 open,finished
	Size       float64 //总数量 多正 空负
	Price      float64 //下单价格
	FillPrice  float64 //成交价格
	Left       float64 //未成交数量
	Role       string  //交易角色 maker,taker
	Iceberg    int64   //冰山数量
	Text       string
	Tif        string //下单类型 gtc:未成交则挂单,ioc:立即成交或者取消，只吃单不挂单,poc:被动委托，只挂单不吃单
	Ordertype  string //订单类型 LIMIT, MARKET, STOP
	CreateTime int64  //创建时间
	UpdateTime int64  //最后更新时间
}
