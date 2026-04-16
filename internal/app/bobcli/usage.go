package bobcli

import "io"

func printUsage(w io.Writer) {
	_, _ = io.WriteString(w, `bob - remote to local browser open bridge

Usage:
  bob <url>
  bob open <url>
  bob doctor
  bob version
  bob tunnel <subcommand>

Environment:
  BOB_ENDPOINT  Forwarded bobd endpoint (default: http://127.0.0.1:17331)
  BOB_TOKEN     Bearer token shared with bobd
  BOB_SESSION   Tunnel/session name used for auto-mirror
  BOB_TIMEOUT   Request timeout (default: 5s)
`)
}

func printTunnelUsage(w io.Writer) {
	_, _ = io.WriteString(w, `Usage:
  bob tunnel up <name> --ssh <target> [--mirror <port>]...
  bob tunnel status [<name>|--all]
  bob tunnel down <name>

Examples:
  bob tunnel up devbox --ssh user@remote-host --mirror 8787
  bob tunnel status devbox
  bob tunnel down devbox
`)
}
