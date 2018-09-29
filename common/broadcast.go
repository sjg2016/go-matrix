// Copyright 2018 The MATRIX Authors as well as Copyright 2014-2017 The go-ethereum Authors
// This file is consisted of the MATRIX library and part of the go-ethereum library.
//
// The MATRIX-ethereum library is free software: you can redistribute it and/or modify it under the terms of the MIT License.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, 
//and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject tothe following conditions:
//
//The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
//
//THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, 
//WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISINGFROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE
//OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
package common

//"github.com/matrix/go-matrix/common"

var (
	broadcastInterval  = uint64(100)
	reelectionInterval = uint64(300)
)

func IsBroadcastNumber(number uint64) bool {
	if number%broadcastInterval == 0 {
		return true
	}
	return false
}

func IsReElectionNumber(number uint64) bool {
	if number%reelectionInterval == 0 {
		return true
	}
	return false
}

func GetLastBroadcastNumber(number uint64) uint64 {
	if IsBroadcastNumber(number) {
		return number
	}
	ans := (number / broadcastInterval) * broadcastInterval
	return ans
}

func GetLastReElectionNumber(number uint64) uint64 {
	if IsReElectionNumber(number) {
		return number
	}
	ans := (number / reelectionInterval) * reelectionInterval
	return ans
}

func GetNextBroadcastNumber(number uint64) uint64 {
	if IsBroadcastNumber(number) {
		return number
	}
	ans := (number/broadcastInterval + 1) * broadcastInterval
	return ans
}

func GetNextReElectionNumber(number uint64) uint64 {
	if IsReElectionNumber(number) {
		return number
	}
	ans := (number/reelectionInterval + 1) * reelectionInterval
	return ans
}

func GetBroadcastInterval() uint64 {
	return broadcastInterval
}
func GetReElectionInterval() uint64 {
	return reelectionInterval
}
