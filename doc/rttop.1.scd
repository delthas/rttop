rttop(1)

# NAME

rttop - *RTT* measuring tool, like *top* for ping

# SYNOPSIS

*rttop* [options...] [server]

# DESCRIPTION

rttop is a simple CLI tool to measure the round-trip time / ping latency of
a connection against a server.

It works like ping, but with advanced real time stats.

rttop can connect to multiple kinds of servers:

- any server that responds to ping (ICMP echo request) message
- servers listening on UDP, running the specific rttop protocol

# OPTIONS

*-h, -help*
	Show help message and quit.

*[server]*
	Server URI (default: "delthas.fr:13770").

	The following URIs are supported:

	- _[udp://]host:port_ connects to a UDP server running the rttop server
	  protocol.
	- _[ping://]host_ connects to a server responding to ping (ICMP echo request)
	  messages. icmp is an alias for the ping scheme.

	If the scheme is omitted, "udp" is assumed if the port is specified, otherwise
	"ping" is assumed.

# AUTHORS

Maintained by delthas <rttop@delthas.fr>, who is assisted by other
open-source contributors. For more information about rttop development, see
<https://github.com/delthas/rttop>.
