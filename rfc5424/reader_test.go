package rfc5424

import (
	"io"
	"strings"
	"testing"
)

func Test_NewReader(t *testing.T) {
	r := NewReader(nil)
	if r == nil {
		t.Fatal("failed to create simple reader")
	}
}

func Test_ReaderSingle(t *testing.T) {
	liner := strings.NewReader("<11>1 sshd is down\n<22>1 sshd is up")

	r := NewReader(liner)
	if r == nil {
		t.Fatal("failed to create simple reader")
	}

	line, err := r.ReadLine()
	if err != nil {
		t.Fatalf("failed to read line: %s", err.Error())
	} else if line != "<11>1 sshd is down" {
		t.Fatalf("read line not correct, got %s, exp %s", line, "<11>1 sshd is down")
	}
}

func Test_ReaderSinglePreceding(t *testing.T) {
	liner := strings.NewReader("xxyyy\n<11>1 sshd is down")

	r := NewReader(liner)
	if r == nil {
		t.Fatal("failed to create simple reader")
	}

	line, err := r.ReadLine()
	if err != nil {
		t.Fatalf("failed to read line: %s", err.Error())
	} else if line != "xxyyy" {
		t.Fatalf("read line not correct, got %s, exp %s", line, "xxyyy")
	}
}

func Test_ReaderEOF(t *testing.T) {
	liner := strings.NewReader("<11>1 sshd is down\n<22>1 sshd is up")

	r := NewReader(liner)
	if r == nil {
		t.Fatal("failed to create simple reader")
	}

	line, err := r.ReadLine()
	if err != nil {
		t.Fatalf("failed to read line: %s", err.Error())
	} else if line != "<11>1 sshd is down" {
		t.Fatalf("read line not correct, got %s, exp %s", line, "<11>1 sshd is down")
	}

	line, err = r.ReadLine()
	if err != io.EOF {
		t.Fatalf("failed to receive EOF as expected")
	}
	if line != "\n<22>1 sshd is up" {
		t.Fatalf("returned line not correct after EOF, got %s", line)
	}
}
