package tcapflow

type DataHandler interface {
	OnData(called_gt SCCPAddress, calling_gt SCCPAddress, data []uint8)
	AfterOnePacket()
	ParseError(data []uint8, recovered interface{})
}

