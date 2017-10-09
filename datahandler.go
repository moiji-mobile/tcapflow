package tcapflow

type DataHandler interface {
	HandleData(called_gt SCCPAddress, calling_gt SCCPAddress, data []uint8)
}

