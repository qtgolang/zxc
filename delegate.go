package amiddleman

type Delegate interface {
	BeforeRequest(entity *Entity)
	BeforeResponse(entity *Entity, err error)
	ErrorLog(err error)
}

// 编辑阶段检查 DefaultDelegate 是否实现了 Delegate 接口
var _ Delegate = &DefaultDelegate{}

type DefaultDelegate struct {
	Delegate
}

func (delegate *DefaultDelegate) BeforeRequest(entity *Entity)             {}
func (delegate *DefaultDelegate) BeforeResponse(entity *Entity, err error) {}
func (delegate *DefaultDelegate) ErrorLog(err error)                       {}
