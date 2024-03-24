package worker

import (
	"flag"
)

func GetArgs() (mode string, localport int, remoteport int) {

	var _mode string
	var _localport int
	var _remoteport int

	flag.StringVar(&_mode, "m", "local", "Mode Local or Remote")
	flag.IntVar(&_localport, "lp", 8080, "Local port on which remote will forwared")
	flag.IntVar(&_remoteport, "rp", 8080, "Remote port which will be forwared")

	flag.Parse()

	return _mode, _localport, _remoteport

}
