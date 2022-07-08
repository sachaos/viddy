package run

type Result struct {
	ID     int64
	stdout []byte
	stderr   []byte
	exitCode int
	err      error
}

