package bitmexadapter

import (
	"testing"
	"github.com/stephenlyu/tds/entity"
	"github.com/Sirupsen/logrus"
	"time"
	"fmt"
)

type _callback struct {
}

func (this _callback) OnTickItem(tick *entity.TickItem) {
	logrus.Infof("%+v", tick)
	//logrus.Infof("Tick: timestamp: %s price: %.04f side: %d open: %.04f buyVolume: %.0f sellVolume: %.0f volume: %.0f", tick.GetDate(), tick.Price, tick.Side, tick.Open, tick.BuyVolume, tick.SellVolume, tick.Volume)
}

func TestBitmexQuoter_Subscribe(t *testing.T) {
	q := NewBitmexQutoterFatory().CreateQuoter(nil)
	q.SetCallback(&_callback{})
	q.Subscribe(entity.ParseSecurityUnsafe("BTCM19.BITMEX"))

	time.Sleep(10 * time.Minute)
	q.Destroy()
}


func TestToSecurity(t *testing.T) {
	fmt.Println(ToSecurity("XBTUSD"))
	fmt.Println(ToSecurity("XBTM19"))
	fmt.Println(FromSecurity(entity.ParseSecurityUnsafe("BTCFUT.BITMEX")))
	fmt.Println(FromSecurity(entity.ParseSecurityUnsafe("BTCM19.BITMEX")))
}
