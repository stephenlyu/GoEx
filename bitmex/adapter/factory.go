package bitmexadapter

import "github.com/stephenlyu/tds/quoter"

type bitmexQuoterFactory struct {
}

func NewBitmexQutoterFatory() quoter.QuoteFactory {
	return &bitmexQuoterFactory{}
}

func (this bitmexQuoterFactory) CreateQuoter(config interface{}) quoter.Quoter {
	return newBitmexQuoter()
}
