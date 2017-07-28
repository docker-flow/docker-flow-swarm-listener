package service

type Sender interface {
	ServicesCreate(services *[]SwarmService, retries, interval int) error
	ServicesRemove(services *[]string, retries, interval int) error
}
