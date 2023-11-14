package futures_wss

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// WsDepthSnapshot
type GateDepthSnapshot struct {
	LastUpdateId int64   `json:"id"`
	Current      float64 `json:"current"`
	Update       float64 `json:"update"`
	Bids         []Bid   `json:"bids"`
	Asks         []Ask   `json:"asks"`
}

func GetDepthSnapshot(symbol, num string) (*GateDepthSnapshot, error) {
	// https://api.gateio.ws/api/v4/futures/usdt/order_book?contract=BTC_USDT&limit=30&with_id=true
	client := http.DefaultClient
	url := fmt.Sprintf(UsdtHttpUrl+"order_book?contract=%s&limit=%s&with_id=true", strings.ToUpper(symbol), num)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bz, err := ioutil.ReadAll(resp.Body)
	var snapshot GateDepthSnapshot
	err = json.Unmarshal(bz, &snapshot)
	if err != nil {
		return nil, err
	}
	if snapshot.Asks == nil && snapshot.Bids == nil {
		return nil, nil
	}
	return &snapshot, nil
}
