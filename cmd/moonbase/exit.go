package main

import "os"

// osExit is a package-level hook for os.Exit. In production, it calls os.Exit directly.
// In tests, it can be replaced with a function that panics with a known value,
// allowing tests to recover and verify exit behavior without terminating the test process.
var osExit = os.Exit
