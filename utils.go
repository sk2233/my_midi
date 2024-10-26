/*
@author: sk
@date: 2024/10/26
*/
package main

import (
	"encoding/binary"
	"io"
	"os"
)

func OpenFile(path string) *os.File {
	file, err := os.Open(path)
	HandleErr(err)
	return file
}

func HandleErr(err error) {
	if err != nil {
		panic(err)
	}
}

// data 必须是指针
func ReadObj(reader io.Reader, data any) {
	err := binary.Read(reader, binary.BigEndian, data)
	HandleErr(err)
}

func ReadBytes(reader io.Reader, cnt int) []byte {
	res := make([]byte, cnt)
	_, err := reader.Read(res)
	HandleErr(err)
	return res
}

func Assert(val bool) {
	if !val {
		panic("assertion failed")
	}
}
