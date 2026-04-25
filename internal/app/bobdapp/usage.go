package bobdapp

import "io"

func printUsage(w io.Writer) {
	_, _ = io.WriteString(w, `bobd - local daemon for bob

Usage:
  bobd serve [--tunnel-name <name> --ssh <target>] [--remote-bob-port 17331] [--local-bobd 127.0.0.1:7331]
  bobd init [--force]
  bobd version

Environment:
  BOBD_BIND            Listen address (default: 127.0.0.1:7331)
  BOBD_TOKEN           Required bearer token
  BOBD_LOCALHOST_ONLY  Allow only loopback URLs (default: true)
`)
}
