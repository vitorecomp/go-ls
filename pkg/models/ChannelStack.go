package models

type OutputChan chan File
type Stack []OutputChan

func (s Stack) Push(channel OutputChan) Stack {
	return append(s, channel)
}

func (s Stack) Pop() (Stack, OutputChan) {
	l := len(s)
	if l == 0 {
		//TODO panic here
	}
	return s[:l-1], s[l-1]
}

func (s Stack) Head() OutputChan {
	l := len(s)
	if l == 0 {
		//TODO panic here
	}
	return s[l-1]
}
