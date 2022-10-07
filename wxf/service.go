package wxf

type Service interface {
	ServiceID() string
	ServiceName() string
	GetMeta() map[string]string
}

type ServiceRegistration interface {
	Service
	PublicAddress() string
	PublicPort() int
	DialURL() string
	GetTags() []string
	GetProtocol() string
	GetNamespace() string
	String() string
}
