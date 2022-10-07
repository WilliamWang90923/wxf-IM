package naming

import "fmt"

type DefaultService struct {
	Id        string
	Name      string
	Address   string
	Port      int
	Protocol  string
	Namespace string
	Tags      []string
	Meta      map[string]string
}

func (s *DefaultService) ServiceID() string {
	return s.Id
}

func (s *DefaultService) ServiceName() string {
	return s.Name
}

func (s *DefaultService) GetMeta() map[string]string {
	return s.Meta
}

func (s *DefaultService) PublicAddress() string {
	return s.Address
}

func (s *DefaultService) PublicPort() int {
	return s.Port
}

func (s *DefaultService) DialURL() string {
	if s.Protocol == "tcp" {
		return fmt.Sprintf("%s:%d", s.Address, s.Port)
	}
	return fmt.Sprintf("%s://%s:%d", s.Protocol, s.Address, s.Port)
}

func (s *DefaultService) GetTags() []string {
	return s.Tags
}

func (s *DefaultService) GetProtocol() string {
	return s.Protocol
}

func (s *DefaultService) GetNamespace() string {
	return s.Namespace
}

func (s *DefaultService) String() string {
	return fmt.Sprintf("Id:%s,Name:%s,Address:%s,Port:%d,Ns:%s,Tags:%v,Meta:%v",
		s.Id, s.Name, s.Address, s.Port, s.Namespace, s.Tags, s.Meta)
}

func NewEntry(id, name, protocol string, address string, port int) *DefaultService {
	return &DefaultService{
		Id:       id,
		Name:     name,
		Address:  address,
		Port:     port,
		Protocol: protocol,
	}
}
