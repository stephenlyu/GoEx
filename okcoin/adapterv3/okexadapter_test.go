package okexadapterv3

import (
	"testing"
	"github.com/stephenlyu/tds/entity"
	"github.com/Sirupsen/logrus"
	"time"
	"github.com/stephenlyu/tds/util"
)

type _callback struct {
}

func (this _callback) OnTickItem(tick *entity.TickItem) {
	//if tick.Volume > 0 {
		logrus.Infof("Tick: code: %s timestamp: %s price: %.04f side: %d open: %.04f buyVolume: %.0f sellVolume: %.0f volume: %.0f", tick.Code, tick.GetDate(), tick.Price, tick.Side, tick.Open, tick.BuyVolume, tick.SellVolume, tick.Volume)
	//}
}

func (this *_callback) OnError(error) {
}

func TestOKExQuoter_Subscribe(t *testing.T) {
	q := NewOKQutoterFatory().CreateQuoter(nil)
	q.SetCallback(&_callback{})
	q.Subscribe(entity.ParseSecurityUnsafe("BTCQFUTUSDT.OKEX"))

	time.Sleep(10 * time.Minute)
	q.Destroy()
}

func TestOKExQuoter_SubscribeIndex(t *testing.T) {
	q := NewOKQutoterFatory().CreateQuoter(nil)
	q.SetCallback(&_callback{})
	q.Subscribe(entity.ParseSecurityUnsafe("BTCINDEX.OKEX"))

	time.Sleep(10 * time.Minute)
	q.Destroy()
}

func TestFromSecurity(t *testing.T) {
	security := entity.ParseSecurityUnsafe("EOSINDEX.OKEX")
	instrumentId := FromSecurity(security)
	security1 := ToSecurity(instrumentId)
	util.Assert(security.String() == security1.String(), "")
}