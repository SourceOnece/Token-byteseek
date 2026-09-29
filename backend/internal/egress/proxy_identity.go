// 连接身份包含地址、认证及状态，用于识别代理连接配置是否变化。
package egress

type ProxyConnectionIdentity struct {
	Protocol string
	Host     string
	Port     int
	Username string
	Password string
	Status   string
}

func ProxyConnectionIdentityFromProxy(proxyIn *Proxy) ProxyConnectionIdentity {
	return ProxyConnectionIdentity{
		Protocol: proxyIn.Protocol,
		Host:     proxyIn.Host,
		Port:     proxyIn.Port,
		Username: proxyIn.Username,
		Password: proxyIn.Password,
		Status:   proxyIn.Status,
	}
}
