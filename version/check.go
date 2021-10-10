package version

var Any = CheckAny{}

type Check interface {
	isVersionCheck()
}

type CheckAny struct{}

func (CheckAny) isVersionCheck() {}

type CheckExact uint64

func (CheckExact) isVersionCheck() {}
