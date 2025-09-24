package cli

import (
	"errors"
	"fmt"
)

// Command represents the high-level action requested by the user.
type Command int

const (
	CommandStart Command = iota
	CommandStatus
	CommandStop
	CommandVersion
)

// Options captures parsed CLI arguments.
type Options struct {
	Command        Command
	Detach         bool
	InternalDaemon bool
	TunnelNames    []string
}

var (
	errStatusWithDetach  = errors.New("status command cannot be used with --detach")
	errStatusWithArgs    = errors.New("status command does not accept tunnel names")
	errStopWithDetach    = errors.New("stop command cannot be used with --detach")
	errStopWithArgs      = errors.New("stop command does not accept tunnel names")
	errVersionWithDetach = errors.New("version command cannot be used with --detach")
	errVersionWithArgs   = errors.New("version command does not accept additional arguments")
)

// Parse inspects the provided arguments and produces structured options.
func Parse(args []string) (*Options, error) {
	opts := &Options{Command: CommandStart}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-d", "--detach":
			if opts.Command == CommandStatus {
				return nil, errStatusWithDetach
			}
			if opts.Command == CommandStop {
				return nil, errStopWithDetach
			}
			if opts.Command == CommandVersion {
				return nil, errVersionWithDetach
			}
			opts.Detach = true
		case "--internal-daemon":
			opts.InternalDaemon = true
		case "status":
			if opts.Command != CommandStart {
				return nil, fmt.Errorf("duplicate command")
			}
			if opts.Detach {
				return nil, errStatusWithDetach
			}
			if len(opts.TunnelNames) > 0 {
				return nil, errStatusWithArgs
			}
			opts.Command = CommandStatus
		case "stop":
			if opts.Command != CommandStart {
				return nil, fmt.Errorf("duplicate command")
			}
			if opts.Detach {
				return nil, errStopWithDetach
			}
			if len(opts.TunnelNames) > 0 {
				return nil, errStopWithArgs
			}
			opts.Command = CommandStop
		case "version":
			if opts.Command != CommandStart {
				return nil, fmt.Errorf("duplicate command")
			}
			if opts.Detach {
				return nil, errVersionWithDetach
			}
			if len(opts.TunnelNames) > 0 {
				return nil, errVersionWithArgs
			}
			opts.Command = CommandVersion
		case "-h", "--help":
			return nil, fmt.Errorf("usage: tunn [--detach|-d] [tunnel ...]\n       tunn status\n       tunn version")
		default:
			if len(arg) > 0 && arg[0] == '-' {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
			if opts.Command == CommandStatus {
				return nil, errStatusWithArgs
			}
			if opts.Command == CommandStop {
				return nil, errStopWithArgs
			}
			if opts.Command == CommandVersion {
				return nil, errVersionWithArgs
			}
			opts.TunnelNames = append(opts.TunnelNames, arg)
		}
	}

	return opts, nil
}
