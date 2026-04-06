package session

import "os"

// stdinFd, stdoutFd, stderrFd return the standard file descriptors.
// Extracted to keep exec calls clean.
func stdinFd() *os.File  { return os.Stdin }
func stdoutFd() *os.File { return os.Stdout }
func stderrFd() *os.File { return os.Stderr }
