package server

import (
	"webtmux/webtty"
)

// Slave is webtty.Slave with some additional methods.
type Slave interface {
	webtty.Slave

	Close() error
}

type Factory interface {
	Name() string
	New(params map[string][]string, headers map[string][]string) (Slave, error)
	// Command returns the command and arguments
	Command() (string, []string)
}
