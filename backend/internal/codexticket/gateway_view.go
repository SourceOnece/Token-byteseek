package codexticket

// VerifiedFlow 只读取当前账号规则，未知配置不能悄悄开启业务通道。
func (s *CodexTicketService) VerifiedFlow(id int64) bool {
	if s == nil {
		return false
	}
	cfg, _ := s.routingConfig(id)
	return cfg != nil && cfg.VerifiedFlow
}
