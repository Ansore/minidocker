package subsystems

// Memory limit, cpu weight, cpu core num
type ResourceConfig struct {
	MemoryLimit string
	CpuShare    string
	CpuSet      string
}

type Subsystem interface {
	Name() string
	Set(path string, res *ResourceConfig) error
	Apply(path string, pid int) error
	Remove(path string) error
}

var (
	SubsystemsIns = []Subsystem{
		&CpusetSubSystem{
      used: false,
    },
		&MemorySubSystem{
      used: false,
    },
		&CpuSubSystem{
      used: false,
    },
	}
)
