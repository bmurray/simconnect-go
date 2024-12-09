# SimConnect API Connector
[![Go Reference](https://pkg.go.dev/badge/github.com/bmurray/bmurray/simconnect-go.svg)](https://pkg.go.dev/github.com/bmurray/simconnect-go)

This package provides a convenient way to interact with the SimConnect API that takes the hassle out of dealing with the polling behavior of the API.

This is based on the seemingly abandoned [msfs2020-go](https://github.com/lian/msfs2020-go) package that implemented vfr map. The critical code is extracted, and a new connector API is layered on top to make writing reliable services much easier. This can be easily integrated with other servies, like UIs, APIs, etc. 

See the [examples](examples) for sample code. The [fuelhack example](examples/fuelhack/) provides the simpliest example of the API. 