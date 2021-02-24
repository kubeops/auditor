package usage

import "fmt"

type LocalInvoiceList struct {
	Items []LocalInvoice
}

type LocalInvoice struct {
	Cluster       string
	Product       string
	Instance      ResourceInstance
	CurrentState  UsageStatus
	PreviousState UsageStatus
}

type ResourceInstance struct {
	Namespace string
	Name      string
	UID       string
}

type UsageStatus struct {
	CPU    string
	Memory string
	Time   int64
}

func (ri ResourceInstance) String() string {
	return fmt.Sprintf("%s/%s, %s", ri.Namespace, ri.Name, ri.UID)
}

func (ru UsageStatus) String() string {
	return fmt.Sprintf("%s cpu, %s memory", ru.CPU, ru.Memory)
}
