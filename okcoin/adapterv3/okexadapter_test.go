package okexadapterv3

import (
	"testing"
	"github.com/stephenlyu/tds/entity"
	"github.com/Sirupsen/logrus"
	"time"
)

type _callback struct {
}

func (this _callback) OnTickItem(tick *entity.TickItem) {
	if tick.Volume > 0 {
		logrus.Infof("Tick: timestamp: %s price: %.04f side: %d open: %.04f buyVolume: %.0f sellVolume: %.0f volume: %.0f", tick.GetDate(), tick.Price, tick.Side, tick.Open, tick.BuyVolume, tick.SellVolume, tick.Volume)
	}
}

func TestOKExQuoter_Subscribe(t *testing.T) {
	q := NewOKQutoterFatory().CreateQuoter(nil)
	q.SetCallback(&_callback{})
	q.Subscribe(entity.ParseSecurityUnsafe("BTCFUT.OKEX"))

	time.Sleep(10 * time.Minute)
	q.Destroy()
}
