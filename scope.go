package vtypes

import "www.velocidex.com/golang/vfilter"

func MakeScope() vfilter.Scope {
	result := vfilter.NewScope()
	result.AddProtocolImpl(GetProtocols()...)

	return result
}
