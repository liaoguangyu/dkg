package dkg

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func IntToByte(num int64)[]byte{
	var buffer bytes.Buffer
	err := binary.Write(&buffer, binary.BigEndian, num)
	if err != nil {
		fmt.Println("IntToByte error")
		fmt.Println(err)
		return nil
	}
	return buffer.Bytes()
}

func ByteToInt(data []byte)int{
	buffer := bytes.NewBuffer(data)
	var temp int
	binary.Read(buffer, binary.BigEndian, &temp)
	return int(temp)
}
