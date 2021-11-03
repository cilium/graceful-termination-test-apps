# graceful-termination-test-apps

A client and a server app that can be used to test graceful terminations of
client to server connections when the server is shut down. When gracefully
terminated, the client exits with zero status code. It panics for failure cases.

A container image containing both can be fetched from
[here](https://hub.docker.com/r/cilium/graceful-termination-test-apps).
