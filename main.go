/*
@author: sk
@date: 2024/10/26
*/
package main

import (
	"fmt"
	"os"
	"strings"
)

// https://hanshuliang.blog.csdn.net/article/details/120883929

const (
	MThdType0 = 0
	MThdType1 = 1
	MThdType2 = 2
)

type MThd struct {
	Mark     [4]byte // 固定为 MThd
	Size     uint32  // 下面数据的大小固定为 6
	Type     uint16  // 0 只有一个轨道  1 有多个轨道且采用同一时间轴  2 多个轨道采用不同时间轴，不常见
	TrackCnt uint16  // 轨道数目 对于类型1一般第一个音轨记录作者等信息，没有演奏信息
	TPQN     uint16  // 最高位标识是 TPQN(每个四分音有多少 tick) 还是 SMPTE 时间格式， 剩下 15位是 每个四分音有多少 tick 一般都是 TPQN 格式且该格式最高位位 0，这里直接认为这个值就是每个四分音有多少 tick
}

type MTrk struct {
	Mark [4]byte // 固定为 MTrk
	Size uint32
	Msgs []*Message
}

const (
	CmdNodeOn       = 1 // 按下
	CmdNodeOff      = 2 // 松开
	CmdDeviceChoose = 3 // 音乐设备选择
	CmdSetControl   = 4 // 混合器参数设置
	CmdPitchBend    = 5 // 弯音
	CmdSystemMsg    = 6 //  系统信息
	CmdNoteAfter    = 7 // 触后音符
	CmdChannelAfter = 8 // 触后通道
	CmdMetaMsg      = 9 // 元数据信息
)

type Message struct {
	DeltaTick uint32 // 变长的编码 间隔tick
	Cmd       uint8
	Channel   uint8
	Data      []byte
}

var (
	QNTime = uint32(0)
)

func main() {
	reader := OpenFile("INTERNET OVERDOSE.mid")
	mthd := &MThd{}
	ReadObj(reader, mthd)
	fmt.Println(mthd)

	offset := float64(0)
	unit := 3.834354 / 1000
	buff := &strings.Builder{}
	for i := uint16(0); i < mthd.TrackCnt; i++ {
		mtrk := ReadMTrk(reader)
		//fmt.Println(mtrk)
		msgs := make([]*Message, 0)
		for _, m := range mtrk.Msgs {
			if m.Cmd == CmdNodeOn || m.Cmd == CmdNodeOff {
				msgs = append(msgs, m)
			}
		}
		if len(msgs) > 0 {
			fmt.Println(len(msgs), i)
		}
		if i == 3 {
			for _, msg := range mtrk.Msgs {
				offset += float64(msg.DeltaTick) * unit
				if msg.Cmd == CmdNodeOn && offset > 0 {
					buff.WriteString(fmt.Sprintf("sleep %f\n", offset))
					buff.WriteString("play 60\n")
					offset = 0
				}
			}
		}
	} // 原来的单位是 微秒
	fmt.Printf("每个 tick %f ms , TPQN %d\n", (float64(QNTime)/float64(mthd.TPQN))/1000, mthd.TPQN)
	fmt.Println(buff.String())
}

func ReadMTrk(reader *os.File) *MTrk {
	mtrk := &MTrk{}
	ReadObj(reader, &mtrk.Mark)
	ReadObj(reader, &mtrk.Size)
	data := ReadBytes(reader, int(mtrk.Size))

	lastStatus := uint8(0)
	deltaTick := uint32(0)
	index := 0
	for index < len(data) {
		deltaTick, index = ParseDeltaTick(data, index)

		status := lastStatus // 若是没有 status 就直接沿用上次的
		if data[index]&0b1000_0000 > 0 {
			status = data[index]
			index++
		}

		switch status & 0b1111_0000 {
		case 0b1001_0000:
			mtrk.Msgs = append(mtrk.Msgs, &Message{DeltaTick: deltaTick, Cmd: CmdNodeOn, Channel: status & 0b1111, Data: data[index : index+2]})
			index += 2
		case 0b1000_0000:
			mtrk.Msgs = append(mtrk.Msgs, &Message{DeltaTick: deltaTick, Cmd: CmdNodeOff, Channel: status & 0b1111, Data: data[index : index+2]})
			index += 2
		case 0b1100_0000:
			mtrk.Msgs = append(mtrk.Msgs, &Message{DeltaTick: deltaTick, Cmd: CmdDeviceChoose, Channel: status & 0b1111, Data: data[index : index+1]})
			index++
		case 0b1101_0000:
			mtrk.Msgs = append(mtrk.Msgs, &Message{DeltaTick: deltaTick, Cmd: CmdChannelAfter, Channel: status & 0b1111, Data: data[index : index+1]})
			index++
		case 0b1010_0000:
			mtrk.Msgs = append(mtrk.Msgs, &Message{DeltaTick: deltaTick, Cmd: CmdNoteAfter, Channel: status & 0b1111, Data: data[index : index+2]})
			index += 2
		case 0b1011_0000:
			mtrk.Msgs = append(mtrk.Msgs, &Message{DeltaTick: deltaTick, Cmd: CmdSetControl, Channel: status & 0b1111, Data: data[index : index+2]})
			index += 2
		case 0b1110_0000: // 两个数据组成高低位
			mtrk.Msgs = append(mtrk.Msgs, &Message{DeltaTick: deltaTick, Cmd: CmdPitchBend, Channel: status & 0b1111, Data: data[index : index+2]})
			index += 2
		case 0b1111_0000: // 系统独占消息
			var msg *Message
			msg, index = ParseSpecialMessage(deltaTick, status, data, index)
			mtrk.Msgs = append(mtrk.Msgs, msg)
		default:
			fmt.Println(status)
		}
		lastStatus = status
	}
	return mtrk
}

func ParseSpecialMessage(deltaTick uint32, status uint8, data []byte, index int) (*Message, int) {
	if status == 0b1111_0000 { // 系统信息消息
		start := index
		for data[index] != 0b1111_0111 {
			index++
		}
		return &Message{DeltaTick: deltaTick, Cmd: CmdSystemMsg, Data: data[start:index]}, index + 1
	} else if status == 0b1111_1111 { // 元数据信息
		l := int(data[index+1]) + 2
		if data[index] == 0x51 { // 一个四分音节的时长  微秒
			QNTime = uint32(data[index+2])<<16 | uint32(data[index+3])<<8 | uint32(data[index+4])
		}
		return &Message{DeltaTick: deltaTick, Cmd: CmdMetaMsg, Data: data[index : index+l]}, index + l
	}
	panic("invalid status")
}

func ParseDeltaTick(data []byte, index int) (uint32, int) { // 使用专门的 Parser 维护指针 还有对应的 Match 方法
	res := uint32(0)
	for data[index]&0b1000_0000 > 0 {
		res = (res << 7) | uint32(data[index]&0b0111_1111)
		index++
	}
	res = (res << 7) | uint32(data[index]&0b0111_1111)
	return res, index + 1
}