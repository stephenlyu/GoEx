package okexadapterv3

import "github.com/stephenlyu/tds/quoter"

type okQuoterFactory struct {
}

func NewOKQutoterFatory() quoter.QuoteFactory {
	return &okQuoterFactory{}
}

func (this okQuoterFactory) CreateQuoter(config interface{}) quoter.Quoter {
	return newOKExQuoter()
}
