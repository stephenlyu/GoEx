package okexadapter

import (
	"testing"
	"github.com/stephenlyu/tds/entity"
	"github.com/Sirupsen/logrus"
	"time"
)

type _callback struct {
}

func (this _callback) OnTickItem(tick *entity.TickItem) {
	logrus.Infof("Tick: timestamp: %d price: %.04f side: %d", tick.Timestamp, tick.Price, tick.Side)
}

func TestOKExQuoter_Subscribe(t *testing.T) {
	q := NewOKQutoterFatory().CreateQuoter(nil)
	q.SetCallback(&_callback{})
	q.Subscribe(entity.ParseSecurityUnsafe("EOSQFUT.OKEX"))

	time.Sleep(1 * time.Minute)
	q.Destroy()
}
