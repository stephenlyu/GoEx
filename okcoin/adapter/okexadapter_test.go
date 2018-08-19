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
	logrus.Infof("Tick: %+v", tick)
}

func TestOKExQuoter_Subscribe(t *testing.T) {
	q := NewOKQutoterFatory().CreateQuoter(nil)
	q.SetCallback(&_callback{})
	q.Subscribe(entity.ParseSecurityUnsafe("EOSQFUT.OKEX"))

	time.Sleep(1 * time.Minute)
	q.Destroy()
}
