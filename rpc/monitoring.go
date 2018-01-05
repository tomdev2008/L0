package rpc

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
)

type Monitoring struct {
}

func NewMonitoring() *Monitoring {
	return &Monitoring{}
}

func (m *Monitoring) QueryLog(args []int, reply *[]byte) error {
	var (
		result  []byte
		err     error
		maxLine = 50
	)
	f := func(line, number int) ([]byte, error) {
		file, err := os.Open(jrpcCfg.LogFilePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		// file.Seek(o, os.SEEK_END)
		r := bufio.NewReader(file)
		var n int
		var result []byte
		for {
			l, isPrefix, err := r.ReadLine()
			if err != nil {
				return nil, err
			}
			if isPrefix {
				result = append(result, l...)
				continue
			}
			if line == 0 {
				l = append(l, '\n')
				result = append(result, l...)
			}
			if line < n && n <= (line+number) {
				l = append(l, '\n')
				result = append(result, l...)
			}
			if n == line+number {
				break
			}
			n++
		}
		return result, nil
	}
	if len(args) == 1 {
		if args[0] < 0 {
			err = errors.New("param need >= 0")
		} else if args[0] > maxLine {
			result, err = f(0, maxLine)
		} else {
			result, err = f(0, args[0])
		}
	} else if len(args) == 2 {
		if args[0] < 0 || (args[1] < 0 && args[1] != -1) {
			err = errors.New("params need >= 0, but the second parans can be -1 to query the last log")
		} else if args[1] == -1 || args[1] > maxLine {
			result, err = f(args[0], maxLine)
		} else {
			result, err = f(args[0], args[1])
		}
	} else {
		err = errors.New("params len not bigger two")
	}
	if err != nil {
		return err
	}
	*reply = result
	return nil
}

func (m *Monitoring) QueryConfig(key string, reply *[]byte) error {
	result, err := ioutil.ReadFile(jrpcCfg.ConfigFilePath)
	if err != nil {
		return err
	}
	*reply = result
	return nil
}
